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
	"context"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3Resource struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	Region          string
	SSL             bool
	Object          string
	objInfoETag     string
}

func (job *Job) HandleS3Resource() (changed bool, err error) {
	c, err := minio.New(job.Resource.S3.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(job.Resource.S3.AccessKeyID, job.Resource.S3.SecretAccessKey, ""),
		Secure: job.Resource.S3.SSL,
	})
	if err != nil {
		return
	}
	ctx := context.Background()
	objInfo, err := c.StatObject(ctx, job.Resource.S3.Bucket, job.Resource.S3.Object, minio.StatObjectOptions{})
	if err != nil {
		return
	}
	if job.Resource.S3.objInfoETag == objInfo.ETag {
		return
	}
	obj, err := c.GetObject(ctx, job.Resource.S3.Bucket, job.Resource.S3.Object, minio.GetObjectOptions{})
	if err != nil {
		return
	}
	defer obj.Close()
	changed, err = job.UpdateFile(obj)
	if err != nil {
		return
	}
	job.Resource.S3.objInfoETag = objInfo.ETag
	return
}
