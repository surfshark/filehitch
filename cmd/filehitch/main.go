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

package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io/fs"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/surfshark/filehitch/internal/config"
	"github.com/surfshark/filehitch/pkg/filehitch"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	//log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	debug := flag.Bool("debug", false, "sets log level to debug")
	configFile := flag.String("config", "config.yaml", "Configuration file")
	flag.Parse()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	cfg, err := config.LoadConfigFile(*configFile)
	if err != nil {
		log.Error().Err(err).Msg("Failed to load config file")
		return
	}
	jobs := make([]filehitch.Job, len(cfg.Jobs))
	for i, jobCfg := range cfg.Jobs {
		jobs[i] = filehitch.Job{
			Name: jobCfg.Name,
			Schedule: filehitch.Schedule{
				Expression: jobCfg.Schedule.Expression,
				Jitter:     jobCfg.Schedule.Jitter,
			},
			Resource: filehitch.Resource{
				Type: jobCfg.Resource.Type,
				Decryption: filehitch.Decryption{
					Password: []byte(jobCfg.Resource.Decryption.Password),
				},
			},
			File: filehitch.File{
				Path: jobCfg.File.Path,
				Permissions: filehitch.Permissions{
					Owner: jobCfg.File.Permissions.Owner,
					Group: jobCfg.File.Permissions.Group,
				},
			},
			Trigger: filehitch.Trigger{
				Command: jobCfg.Trigger.Command,
				User:    jobCfg.Trigger.User,
				Group:   jobCfg.Trigger.Group,
				CWD:     jobCfg.Trigger.CWD,
			},
		}
		jobs[i].Schedule.Timezone, err = time.LoadLocation(jobCfg.Schedule.Timezone)
		if err != nil {
			log.Error().Err(err).Str("job", jobs[i].Name).Msg("Failed to load timezone")
			return
		}
		switch jobs[i].Resource.Type {
		case "http":
			jobs[i].Resource.HTTP = &filehitch.HTTPResource{
				URL:     jobCfg.Resource.HTTP.URL,
				Method:  jobCfg.Resource.HTTP.Method,
				Headers: jobCfg.Resource.HTTP.Headers,
				Expect: filehitch.Expect{
					Code: jobCfg.Resource.HTTP.Expect.Code,
				},
				Timeout:     jobCfg.Resource.HTTP.Timeout,
				MaxAttempts: jobCfg.Resource.HTTP.MaxAttempts,
			}
			jobs[i].Resource.HTTP.Body, err = base64.StdEncoding.DecodeString(jobCfg.Resource.HTTP.Body)
			if err != nil {
				log.Error().Err(err).Str("job", jobs[i].Name).Msg("Failed to decode base64 HTTP resource request body")
				return
			}
		case "s3":
			jobs[i].Resource.S3 = &filehitch.S3Resource{
				Endpoint:        jobCfg.Resource.S3.Endpoint,
				AccessKeyID:     jobCfg.Resource.S3.AccessKeyID,
				SecretAccessKey: jobCfg.Resource.S3.SecretAccessKey,
				Bucket:          jobCfg.Resource.S3.Bucket,
				SSL:             jobCfg.Resource.S3.SSL,
				Region:          jobCfg.Resource.S3.Region,
				Object:          jobCfg.Resource.S3.Object,
				// TODO: Implement timeout and max attempts.
				//Timeout:         jobCfg.Resource.S3.Timeout,
				//MaxAttempts:     jobCfg.Resource.S3.MaxAttempts,
			}
		}
		jobs[i].File.Permissions.Mode, err = stringToFileMode(jobCfg.File.Permissions.Mode)
		if err != nil {
			log.Error().Err(err).Str("job", jobs[i].Name).Msg("Failed to convert string to file mode")
			return
		}
	}
	cfg = &config.Config{}
	filehitch.ScheduleJobs(jobs)
	select {}
}

func stringToFileMode(str string) (fs.FileMode, error) {
	modeInt, err := strconv.ParseUint(str, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to convert string to file mode: %w", err)
	}
	return fs.FileMode(modeInt), nil
}
