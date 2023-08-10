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
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

type HTTPResource struct {
	URL                   string
	Method                string
	Headers               map[string][]string
	Expect                Expect
	Timeout               time.Duration
	Body                  []byte
	MaxAttempts           int
	headerIfNoneMatch     string
	headerIfModifiedSince string
}

type Expect struct {
	Code int
}

func (job *Job) HandleHTTPResource() (changed bool, err error) {
	c := &http.Client{
		Timeout: job.Resource.HTTP.Timeout,
	}
	for attempt := 1; attempt <= job.Resource.HTTP.MaxAttempts; attempt++ {
		var req *http.Request
		req, err = http.NewRequest(job.Resource.HTTP.Method, job.Resource.HTTP.URL, bytes.NewBuffer(job.Resource.HTTP.Body))
		if err != nil {
			return
		}
		if job.Resource.HTTP.headerIfNoneMatch != "" {
			req.Header.Add("If-None-Match", job.Resource.HTTP.headerIfNoneMatch)
		}
		// If-Modified-Since can only be used with a GET or HEAD.
		// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/If-Modified-Since
		if job.Resource.HTTP.Method == http.MethodGet || job.Resource.HTTP.Method == http.MethodHead {
			if job.Resource.HTTP.headerIfModifiedSince != "" {
				req.Header.Add("If-Modified-Since", job.Resource.HTTP.headerIfModifiedSince)
			}
		}
		job.Resource.HTTP.AddHeaders(req)
		var resp *http.Response
		resp, err = c.Do(req)
		if err != nil {
			if attempt < job.Resource.HTTP.MaxAttempts {
				log.Warn().Str("job", job.Name).Int("attempt", attempt).Int("max_attempts", job.Resource.HTTP.MaxAttempts).Err(err).Msg("Failed HTTP request, retrying")
				time.Sleep(time.Second)
				continue
			}
			log.Error().Str("job", job.Name).Int("attempt", attempt).Int("max_attempts", job.Resource.HTTP.MaxAttempts).Err(err).Msg("Failed HTTP request")
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusNotModified {
			return
		}
		if resp.StatusCode != job.Resource.HTTP.Expect.Code {
			err = fmt.Errorf("unexpected HTTP status code %d, expected %d", resp.StatusCode, job.Resource.HTTP.Expect.Code)
			if attempt < job.Resource.HTTP.MaxAttempts {
				log.Warn().Str("job", job.Name).Int("attempt", attempt).Int("max_attempts", job.Resource.HTTP.MaxAttempts).Err(err).Msg("Unexpected HTTP status code, retyring")
				time.Sleep(time.Second)
				continue
			}
			log.Error().Str("job", job.Name).Int("attempt", attempt).Int("max_attempts", job.Resource.HTTP.MaxAttempts).Err(err).Msg("Unexpected HTTP status code")
			return
		}
		changed, err = job.UpdateFile(resp.Body)
		if err != nil {
			return
		}
		job.Resource.HTTP.headerIfNoneMatch = resp.Header.Get("ETag")
		// If-Modified-Since can only be used with a GET or HEAD.
		// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/If-Modified-Since
		if job.Resource.HTTP.Method == http.MethodGet || job.Resource.HTTP.Method == http.MethodHead {
			job.Resource.HTTP.headerIfModifiedSince = resp.Header.Get("Date")
		}
		return
	}
	return
}

func (resource *HTTPResource) AddHeaders(req *http.Request) {
	for key, values := range resource.Headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
}
