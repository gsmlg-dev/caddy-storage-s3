package caddy_storage_s3

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/certmagic"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
)

func init() {
	caddy.RegisterModule(S3Storage{})
}

type S3Storage struct {
	logger *zap.Logger
	Client *minio.Client

	muLocks *sync.RWMutex
	locks   map[string]string

	Host      string `json:"host"`
	Bucket    string `json:"bucket"`
	AccessID  string `json:"access_id"`
	SecretKey string `json:"secret_key"`
	Prefix    string `json:"prefix"`
	Insecure  bool   `json:"insecure"`
}

// CaddyModule returns the Caddy module information.
func (S3Storage) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "caddy.storage.s3",
		New: func() caddy.Module {
			s := S3Storage{
				locks:   make(map[string]string),
				muLocks: &sync.RWMutex{},
			}
			return &s
		},
	}
}

// CertMagicStorage converts s to a certmagic.Storage instance.
func (s *S3Storage) CertMagicStorage() (certmagic.Storage, error) {
	return s, nil
}

// UnmarshalCaddyfile sets up the storage module from Caddyfile tokens.
func (s *S3Storage) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		var value string

		key := d.Val()

		if !d.Args(&value) {
			continue
		}

		switch key {
		case "host":
			s.Host = value
		case "bucket":
			s.Bucket = value
		case "access_id":
			s.AccessID = value
		case "secret_key":
			s.SecretKey = value
		case "prefix":
			s.Prefix = value
		case "insecure":
			insecure, err := strconv.ParseBool(value)
			if err != nil {
				return d.Err("Invalid usage of insecure in s3-storage config: " + err.Error())
			}
			s.Insecure = insecure
		}
	}

	return nil
}

func (s *S3Storage) Provision(ctx caddy.Context) error {
	s.logger = ctx.Logger(s)

	// Load Environment
	if s.Host == "" {
		s.Host = os.Getenv("S3_HOST")
	}

	if s.Bucket == "" {
		s.Bucket = os.Getenv("S3_BUCKET")
	}

	if s.AccessID == "" {
		s.AccessID = os.Getenv("S3_ACCESS_ID")
	}

	if s.SecretKey == "" {
		s.SecretKey = os.Getenv("S3_SECRET_KEY")
	}

	if s.Prefix == "" {
		s.Prefix = os.Getenv("S3_PREFIX")
	}

	if !s.Insecure {
		insecure := os.Getenv("S3_INSECURE")
		if insecure != "" {
			s.Insecure, _ = strconv.ParseBool(insecure)
		}
	}
	secure := !s.Insecure

	// minio Client
	client, err := minio.New(s.Host, &minio.Options{
		Creds:  credentials.NewStaticV4(s.AccessID, s.SecretKey, ""),
		Secure: secure,
	})

	if err != nil {
		s.logger.Debug(fmt.Sprintf("Provision error: %v", err))
		return err
	} else {
		s.logger.Debug(fmt.Sprintf("Provision client: %v", client))
		s.Client = client

		list, err := s.List(ctx, s.Prefix, true)
		s.logger.Debug(fmt.Sprintf("Provision client list: %v\nerror: %v", list, err))
	}

	return nil
}

func (s *S3Storage) Validate() error {
	return nil
}

func (s *S3Storage) Exists(ctx context.Context, key string) bool {
	key = s.KeyPrefix(key)

	_, err := s.Client.StatObject(ctx, s.Bucket, key, minio.StatObjectOptions{})

	exists := err == nil

	s.logger.Debug(fmt.Sprintf("Check exists: %s, %t", key, exists))

	return exists
}

func (s3 *S3Storage) Store(ctx context.Context, key string, value []byte) error {
	key = s3.KeyPrefix(key)
	length := int64(len(value))

	s3.logger.Debug(fmt.Sprintf("Store: %s, %d bytes", key, length))

	_, err := s3.Client.PutObject(ctx, s3.Bucket, key, bytes.NewReader(value), length, minio.PutObjectOptions{})

	return err
}

func (s3 *S3Storage) Load(ctx context.Context, key string) ([]byte, error) {
	key = s3.KeyPrefix(key)

	s3.logger.Debug(fmt.Sprintf("Load key: %s", key))

	object, err := s3.Client.GetObject(ctx, s3.Bucket, key, minio.GetObjectOptions{})

	if err != nil {
		s3.logger.Debug(fmt.Sprintf("Load key: %s; Error %v", key, err))
		return nil, err
	}
	s3.logger.Debug(fmt.Sprintf("Load key: %s; object %v", key, object))
	b, err := ioutil.ReadAll(object)
	if err != nil {
		s3.logger.Debug(fmt.Sprintf("Load key: %s; ioutil.ReadAll Error %v", key, err))
		return b, fs.ErrNotExist
	}
	return b, nil
}

func (s3 *S3Storage) Delete(ctx context.Context, key string) error {
	key = s3.KeyPrefix(key)

	s3.logger.Debug(fmt.Sprintf("Delete key: %s", key))

	return s3.Client.RemoveObject(ctx, s3.Bucket, key, minio.RemoveObjectOptions{})
}

func (s3 *S3Storage) List(ctx context.Context, prefix string, recursive bool) ([]string, error) {

	objects := s3.Client.ListObjects(ctx, s3.Bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: recursive,
	})

	keys := make([]string, len(objects))

	for object := range objects {
		keys = append(keys, object.Key)
	}

	s3.logger.Debug(fmt.Sprintf("List objects: %v", keys))

	return keys, nil
}

func (s3 *S3Storage) Stat(ctx context.Context, key string) (certmagic.KeyInfo, error) {
	key = s3.KeyPrefix(key)

	object, err := s3.Client.StatObject(ctx, s3.Bucket, key, minio.StatObjectOptions{})

	if err != nil {
		s3.logger.Error(fmt.Sprintf("Stat key: %s, error: %v", key, err))

		return certmagic.KeyInfo{}, nil
	}

	s3.logger.Debug(fmt.Sprintf("Stat key: %s, size: %d bytes", key, object.Size))

	return certmagic.KeyInfo{
		Key:        object.Key,
		Modified:   object.LastModified,
		Size:       object.Size,
		IsTerminal: strings.HasSuffix(object.Key, "/"),
	}, err
}

func (s *S3Storage) Filename(key string) string {
	return s.KeyPrefix(key)
}

func (s *S3Storage) KeyPrefix(key string) string {
	return s.Prefix + "/" + key
}

func (s *S3Storage) Lock(ctx context.Context, key string) error {
	key = s.KeyPrefix(key)
	s.logger.Debug(fmt.Sprintf("Lock: %s", key))

	s.muLocks.Lock()
	s.locks[key] = key
	s.muLocks.Unlock()

	return nil
}

func (s *S3Storage) Unlock(ctx context.Context, key string) error {
	key = s.KeyPrefix(key)
	s.logger.Debug(fmt.Sprintf("Unlock: %s", key))

	s.muLocks.Lock()
	delete(s.locks, key)
	s.muLocks.Unlock()

	return nil
}

func (s *S3Storage) String() string {
	return "S3Storage:" + s.Host + ":" + s.Bucket + ":" + s.Prefix
}

// Interface guards
var (
	_ caddy.StorageConverter = (*S3Storage)(nil)
	_ caddyfile.Unmarshaler  = (*S3Storage)(nil)
	_ certmagic.Storage      = (*S3Storage)(nil)
)
