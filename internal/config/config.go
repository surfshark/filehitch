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

package config

import (
	"os"
	"time"

	"github.com/creasty/defaults"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Jobs []Job `yaml:"jobs"`
}

type Job struct {
	Name     string   `yaml:"name"`
	Schedule Schedule `yaml:"schedule"`
	Resource Resource `yaml:"resource"`
	File     File     `yaml:"file"`
	Trigger  Trigger  `yaml:"trigger"`
}

type Schedule struct {
	Expression string `yaml:"expression"`
	Timezone   string `yaml:"timezone" default:"Local"`
	Jitter     int    `yaml:"jitter" default:"5"`
}

type Resource struct {
	Type       string        `yaml:"type"`
	Decryption Decryption    `yaml:"decryption"`
	HTTP       *HTTPResource `yaml:"http"`
	S3         *S3Resource   `yaml:"s3"`
}

type HTTPResource struct {
	URL         string              `yaml:"url"`
	Method      string              `yaml:"method"`
	Headers     map[string][]string `yaml:"headers"`
	Expect      Expect              `yaml:"expect"`
	Timeout     time.Duration       `yaml:"timeout"`
	MaxAttempts int                 `yaml:"max_attempts" default:"3"`
	Body        string              `yaml:"body"`
}

type S3Resource struct {
	Endpoint        string `yaml:"endpoint"`
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
	Bucket          string `yaml:"bucket"`
	Region          string `yaml:"region"`
	SSL             bool   `yaml:"ssl" default:"true"`
	Object          string `yaml:"object"`
}

type Expect struct {
	Code int    `yaml:"code" default:"200"`
	Body string `yaml:"body"`
}

type Decryption struct {
	Password string `yaml:"password"`
}

type File struct {
	Path        string      `yaml:"path"`
	Permissions Permissions `yaml:"permissions"`
}

type Permissions struct {
	Mode  string `yaml:"mode" default:"0644"`
	Owner string `yaml:"owner" default:"root"`
	Group string `yaml:"group" default:"root"`
}

type Trigger struct {
	Command []string `yaml:"command"`
	User    string   `yaml:"user" default:"root"`
	Group   string   `yaml:"group" default:"root"`
	CWD     string   `yaml:"cwd"`
}

func LoadConfigFile(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	err = defaults.Set(&cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
