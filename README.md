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
