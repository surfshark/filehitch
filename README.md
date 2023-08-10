# Filehitch

> Automate remote file synchronization by monitoring changes in HTTP and S3 resources based on a specified cron interval. Decrypt and trigger commands as needed.

[![Github release version](https://img.shields.io/github/v/release/surfshark/filehitch.svg?include_prereleases)](https://github.com/surfshark/filehitch/releases/latest)
[![Go report](https://goreportcard.com/badge/github.com/surfshark/filehitch)](https://goreportcard.com/report/github.com/surfshark/filehitch)
[![GoDoc](https://godoc.org/github.com/surfshark/filehitch?status.svg)](https://godoc.org/github.com/surfshark/filehitch)
[![License](https://img.shields.io/github/license/surfshark/filehitch.svg)](https://github.com/surfshark/filehitch/blob/master/LICENSE)
[![Code with hearth by Stnby](https://img.shields.io/badge/%3C%2F%3E%20with%20%E2%99%A5%20by-Stnby-ff1414.svg)](https://github.com/stnby)

## Install
To install the latest version of Filehitch from sources, run:
```sh
go install github.com/surfshark/filehitch/cmd/filehitch@latest
```

To install [Secure IO](https://github.com/minio/sio#readme) file encryption tool from sources, run:
```sh
go install github.com/minio/sio/cmd/ncrypt@latest
```

## Features
* **Cron Expression Scheduling**: Set up automated monitoring intervals using precise cron expressions, including seconds, minutes, hours, days of the month, months, days of the week, and multiple timezones support.
* **HTTP and S3 Resource Monitoring**: Monitor remote files via HTTP endpoints and S3 resources, ensuring you stay updated with the latest changes.
* **Secure Encryption**: Utilize the robust ChaCha20-Poly1305 algorithm for seamless decryption of encrypted resources, ensuring secure synchronization and access to sensitive data.
* **Customizable Command Triggers**: Execute custom commands of your choice when changes in remote files are detected, enabling flexible automation.
* **Flexible Configuration**: Intuitive YAML configuration for easy setup and maintenance.
* **User-Friendly Setup**: Intuitive YAML configuration allows for easy and efficient setup and maintenance of the synchronization process.

## Configuration
```yaml
jobs:
  # Define a job named "HTTP example job"
  - name: HTTP example job
    schedule:
      # Set the cron expression for the job to run every hour
      expression: "0 0 * * * *"
      # Specify the timezone for scheduling (America/New_York)
      timezone: America/New_York
      # Introduce a jitter of up to 10 minutes to prevent synchronized execution
      jitter: 600
    resource:
      # Specify the resource type as HTTP for remote file monitoring
      type: http
      http:
        # Specify the URL of the remote file
        url: https://example.com/your-remote-file.txt
        # Specify the HTTP method (GET, POST, etc.)
        method: "GET"
        # Specify custom headers if needed
        headers:
          User-Agent:
            - "MyApp/1.0"
          Authorization:
            - "Bearer YOUR_TOKEN"
        # Specify the expected HTTP response code
        expect:
          code: 200
        # Set a timeout for the HTTP request
        timeout: 30s
        # Optionally include a request body, use base64 encoded string
        body: SGVsbG8sIFdvcmxkIQo=
        # Set the maximum number of attempts
        max_attempts: 3
    file:
      # Specify the local file path to save the downloaded file
      path: "/home/alice/your-other-file-path.txt"
      permissions:
        # Set file permissions to read and write for the owner only
        mode: "0600"
        # Specify the owner of the file
        owner: "alice"
        # Specify the group of the file
        group: "alice"
    trigger:
      # Define the command to be triggered upon file change detection
      command: ["/home/alice/scripts/run.sh", "--input", "your-other-file-path.txt"]
      # Specify the user for executing the trigger command
      user: "alice"
      # Specify the group for executing the trigger command
      group: "alice"
      # Set the current working directory for the trigger command
      cwd: "/home/alice"

  # Define a job named "S3 example job"
  - name: S3 example job
    schedule:
      # Set the cron expression for the job to run every 30 seconds
      expression: "*/30 * * * * *"
      # Specify the timezone for scheduling (Europe/Amsterdam)
      timezone: Europe/Amsterdam
      # Introduce a jitter of up to 5 seconds to prevent synchronized execution
      jitter: 5 
    resource:
      # Specify the resource type as S3 for remote file monitoring
      type: s3
      s3:
        # Specify the S3 endpoint (for AWS, use s3.amazonaws.com)
        endpoint: s3.amazonaws.com
        # Specify the AWS region for the S3 bucket
        region: your-aws-region
        # Specify your AWS access key ID
        access_key_id: YOUR_AWS_ACCESS_KEY_ID
        # Specify your AWS secret access key
        secret_access_key: YOUR_AWS_SECRET_ACCESS_KEY
        # Specify the name of the S3 bucket
        bucket: my-s3-bucket
        # Specify the path to the object within the bucket
        object: your-file-path.txt.enc
      decryption:
        # Specify the decryption password only if the object is encrypted
        password: "YourDecryptionPassword"
    file:
      # Specify the local file path to save the downloaded file
      path: "/home/alice/your-file-path.txt"
      permissions:
        # Set file permissions to read and write for the owner only
        mode: "0600"
        # Specify the owner of the file
        owner: "alice"
        # Specify the group of the file
        group: "alice"
    trigger:
      # Define the command to be triggered upon file change detection
      command: ["/home/alice/scripts/run.sh", "--input", "your-file-path.txt"]
      # Specify the user for executing the trigger command
      user: "alice"
      # Specify the group for executing the trigger command
      group: "alice"
      # Set the current working directory for the trigger command
      cwd: "/home/alice"
```

## Encrypting a file
Make sure [Secure IO](https://github.com/minio/sio#readme) file encryption tool is installed and run:
```sh
ncrypt -cipher C20P1305 your-file.txt > your-file.txt.enc
```

## License
This project is licensed under the Apache License, Version 2.0 - see the [LICENSE](https://github.com/surfshark/filehitch/blob/master/LICENSE) file for details.
