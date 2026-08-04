## Caddy-Storage-S3

Caddy S3-compatible storage driver(minio).

### Guide

Build

    go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest

    xcaddy build master --output ./caddy --with github.com/gsmlg-dev/caddy-storage-s3

Build container

    FROM caddy:builder AS builder
    RUN xcaddy build master --with github.com/gsmlg-dev/caddy-storage-s3 

    FROM caddy
    COPY --from=builder /usr/bin/caddy /usr/bin/caddy

Run

    caddy run --config caddy.json

Caddyfile Example

    # Global Config

    {
        storage s3 {
            host "Host"
            bucket "Bucket"
            access_id "Access ID"
            secret_key "Secret Key"
            prefix "ssl"
            insecure false #disables SSL if true
        }
    }

JSON Config Example

    {
      "storage": {
        "module": "s3",
        "host": "Host",
        "bucket": "Bucket",
        "access_id": "Access ID",
        "secret_key": "Secret Key",
        "prefix": "ssl",
        "insecure": false
      }
      "app": {
        ...
      }
    }

From Environment

    S3_HOST
    S3_BUCKET
    S3_ACCESS_ID
    S3_SECRET_KEY
    S3_PREFIX
    S3_INSECURE

## Authentication

Credentials are resolved in the following priority order:

1. **Static credentials** — explicitly configured in the Caddyfile/JSON
   (`access_id` / `secret_key`) or via the `S3_ACCESS_ID` / `S3_SECRET_KEY`
   environment variables.
2. **AWS provider chain** — only used when no static credentials are set:
   - `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` / `AWS_SESSION_TOKEN`
     environment variables.
   - **IAM role** (EC2 instance profile / ECS / EKS). Credentials are fetched
     from the instance metadata service and refreshed automatically when they
     expire.
   - Shared credentials file (`~/.aws/credentials`).

### Using an EC2 IAM role (recommended)

To let Caddy use the instance role credentials automatically, **do not** set
`S3_ACCESS_ID` / `S3_SECRET_KEY` (leave them empty) and make sure the bucket /
endpoint are reachable with the role's permissions:

    S3_HOST=s3.us-east-1.amazonaws.com
    S3_BUCKET=my-cert-bucket

> **Note:** IAM role credentials are temporary and include a session token.
> Do not copy them into `S3_ACCESS_ID` / `S3_SECRET_KEY` — the static path does
> not handle the session token and the credentials would expire.

#### IMDSv2 + Docker (important)

When running in a container on EC2 with the instance metadata service v2
(`HttpTokens=required`), the default **hop limit of 1** causes metadata requests
from inside a container to be dropped. Fix it with one of:

- Launch the instance with `--metadata-options HttpPutResponseHopLimit=2`, or
- Run the container with `--network host`.
