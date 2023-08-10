// Copyright 2023 Laurynas Četyrkinas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package filehitch

import (
	"bytes"
	"crypto/sha512"
	"fmt"
	"io"
	"io/fs"
	"math/rand"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"syscall"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/minio/sio"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/scrypt"
)

type Job struct {
	Name     string
	Schedule Schedule
	Resource Resource
	File     File
	Trigger  Trigger
}

type Schedule struct {
	Expression string
	Timezone   *time.Location
	Jitter     int
}

type Resource struct {
	Type       string
	Decryption Decryption
	HTTP       *HTTPResource
	S3         *S3Resource
}

type Decryption struct {
	Password []byte
}

type File struct {
	Path        string
	Permissions Permissions
}

type Permissions struct {
	Mode  fs.FileMode
	Owner string
	Group string
}

type Trigger struct {
	Command []string
	User    string
	Group   string
	CWD     string
}

func ScheduleJobs(jobs []Job) {
	// Store schedulers in a map with timezone strings as keys,
	// as each scheduler has its own timezone setting.
	scheds := make(map[string]*gocron.Scheduler)
	for i := 0; i < len(jobs); i++ {
		job := &jobs[i]
		tzname := job.Schedule.Timezone.String()
		sched, ok := scheds[tzname]
		if !ok {
			sched = gocron.NewScheduler(job.Schedule.Timezone)
			scheds[tzname] = sched
			log.Debug().Str("job", job.Name).Str("scheduler", tzname).Msg("Created scheduler")
		}
		sched.CronWithSeconds(job.Schedule.Expression).Do(job.Run)
		log.Debug().Str("job", job.Name).Str("scheduler", tzname).Str("expression", job.Schedule.Expression).Msg("Added cron job to scheduler")
	}
	for tzname, sched := range scheds {
		sched.StartAsync()
		log.Debug().Str("scheduler", tzname).Msg("Started scheduler")
	}
}

func (job *Job) Run() {
	if job.Schedule.Jitter > 0 {
		jitter := time.Millisecond * time.Duration(rand.Intn(int(job.Schedule.Jitter)*1000))
		log.Debug().Str("job", job.Name).Dur("jitter", jitter).Msg("Delaying due to jitter")
		time.Sleep(jitter)
	}
	log.Info().Str("job", job.Name).Msg("Starting job")
	var (
		updated bool
		err     error
	)
	switch job.Resource.Type {
	case "http":
		updated, err = job.HandleHTTPResource()
	case "s3":
		updated, err = job.HandleS3Resource()
	}
	if err != nil {
		log.Error().Str("job", job.Name).Err(err).Msg("Failed job")
		return
	}
	if !updated {
		log.Info().Str("job", job.Name).Msg("Finished job, file not updated")
		return
	}
	if len(job.Trigger.Command) > 0 {
		err = job.ExecuteTrigger()
		if err != nil {
			log.Error().Str("job", job.Name).Err(err).Msg("Failed to execute trigger")
		}
	}
	log.Info().Str("job", job.Name).Msg("Finished job, file updated")
}

func (job *Job) UpdateFile(src io.ReadCloser) (changed bool, err error) {
	tempFile, err := CreateTempFile()
	if err != nil {
		return
	}
	name := tempFile.Name()
	log.Debug().Str("job", job.Name).Str("file", name).Msg("Created temporary file")
	defer tempFile.Close()

	var tempHash []byte
	if len(job.Resource.Decryption.Password) > 0 {
		tempHash, err = DecryptWriteToFileAndChecksum(tempFile, src, job.Resource.Decryption.Password)
	} else {
		tempHash, err = WriteToFileAndChecksum(tempFile, src)
	}
	if err != nil {
		return
	}
	currHash, err := job.CalculateFileChecksum()
	if err != nil {
		if os.IsNotExist(err) {
			err = job.FinalizePlacement(tempFile)
			if err == nil {
				changed = true
			}
			return
		}
	}
	if !bytes.Equal(tempHash, currHash) {
		err = job.FinalizePlacement(tempFile)
		if err == nil {
			changed = true
		}
		return
	}
	err = os.Remove(name)
	if err == nil {
		log.Debug().Str("job", job.Name).Str("file", name).Msg("Removed temporary file")
	}
	return
}

func CreateTempFile() (file *os.File, err error) {
	file, err = os.CreateTemp("", "rrchanged-")
	if err != nil {
		return
	}
	return file, os.Chmod(file.Name(), 0600)
}

func DecryptWriteToFileAndChecksum(file *os.File, src io.Reader, password []byte) (sum []byte, err error) {
	salt := make([]byte, 32)
	if _, err = io.ReadFull(src, salt); err != nil {
		err = fmt.Errorf("failed to read salt: %w", err)
		return
	}
	key, err := scrypt.Key(password, salt, 32768, 16, 1, 32)
	if err != nil {
		err = fmt.Errorf("failed to derive key from password and salt: %w", err)
		return
	}
	cfg := sio.Config{Key: key, CipherSuites: []byte{sio.CHACHA20_POLY1305}}
	h := sha512.New()
	_, err = sio.Decrypt(io.MultiWriter(file, h), src, cfg)
	if err != nil {
		err = fmt.Errorf("failed to decrypt: %w", err)
		return
	}
	sum = h.Sum(nil)
	return
}

func WriteToFileAndChecksum(file *os.File, src io.Reader) (sum []byte, err error) {
	h := sha512.New()
	_, err = io.Copy(io.MultiWriter(file, h), src)
	sum = h.Sum(nil)
	return
}

func (job *Job) CalculateFileChecksum() (sum []byte, err error) {
	f, err := os.Open(job.File.Path)
	if err != nil {
		return
	}
	defer f.Close()
	h := sha512.New()
	_, err = io.Copy(h, f)
	sum = h.Sum(nil)
	return
}

func (job *Job) FinalizePlacement(tempFile *os.File) (err error) {
	name := tempFile.Name()
	tempFile.Close()
	err = os.Rename(name, job.File.Path)
	if err != nil {
		return
	}
	u, err := user.Lookup(job.File.Permissions.Owner)
	if err != nil {
		return
	}
	g, err := user.LookupGroup(job.File.Permissions.Group)
	if err != nil {
		return
	}
	uid, _ := strconv.Atoi(u.Uid)
	gid, _ := strconv.Atoi(g.Gid)
	err = os.Chown(job.File.Path, uid, gid)
	if err != nil {
		return
	}
	return os.Chmod(job.File.Path, job.File.Permissions.Mode)
}

func (job *Job) ExecuteTrigger() (err error) {
	cmd := exec.Command(job.Trigger.Command[0], job.Trigger.Command[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = job.Trigger.CWD
	u, err := user.Lookup(job.Trigger.User)
	if err != nil {
		return
	}
	g, err := user.LookupGroup(job.Trigger.Group)
	if err != nil {
		return
	}
	uid, _ := strconv.Atoi(u.Uid)
	gid, _ := strconv.Atoi(g.Gid)
	log.Debug().Str("user", job.Trigger.User).Str("group", job.Trigger.Group).Str("cwd", job.Trigger.CWD).Strs("command", job.Trigger.Command).Msg("Executing trigger command")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: uint32(uid), Gid: uint32(gid),
		},
	}
	return cmd.Run()
}
