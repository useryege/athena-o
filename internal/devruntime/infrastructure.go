package devruntime

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/redis/go-redis/v9"
	"os"
	"path/filepath"
	"time"
)

func (m *Manager) infrastructureSecret(name string) (string, error) {
	var value []byte
	err := withLock(m.Key, func() error {
		path := filepath.Join(m.Key.Dir(), name)
		var err error
		value, err = os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			raw := make([]byte, 32)
			if _, err = rand.Read(raw); err != nil {
				return err
			}
			value = []byte(hex.EncodeToString(raw))
			return atomicFile(path, value)
		}
		return err
	})
	if err != nil {
		return "", err
	}
	if len(value) != 64 {
		return "", errors.New("invalid infrastructure secret")
	}
	return string(value), nil
}
func (m *Manager) prepareAPIInfrastructure(ctx context.Context, env map[string]string) error {
	lease, err := acquireOperation(m.Key)
	if err != nil {
		return err
	}
	defer lease.close()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	password, err := m.infrastructureSecret("redis-password")
	if err != nil {
		return err
	}
	config := []byte("bind 0.0.0.0\nprotected-mode yes\nappendonly yes\nrequirepass " + password + "\n")
	if err = m.SaveSecret("redis.conf", config); err != nil {
		return err
	}
	redisResource, err := m.ensureInfrastructureContainer(ctx, "redis", "redis:7.2.14-alpine", "/data", "6379", []string{"--user", "0:0", "--entrypoint", "redis-server", "--mount", "type=bind,source=" + filepath.Join(m.Key.Dir(), "redis.conf") + ",target=/usr/local/etc/redis/redis.conf,readonly"}, []string{"/usr/local/etc/redis/redis.conf"})
	if err != nil {
		return err
	}
	redisAddress, err := m.containerAddress(ctx, redisResource, "6379/tcp")
	if err != nil {
		return err
	}
	client := redis.NewClient(&redis.Options{Addr: redisAddress, Password: password})
	defer client.Close()
	for {
		if err = client.Ping(ctx).Err(); err == nil {
			break
		}
		select {
		case <-ctx.Done():
			return errors.New("instance Redis readiness failed")
		case <-time.After(100 * time.Millisecond):
		}
	}
	env["REDIS_SERVER"] = redisAddress
	env["REDIS_PASSWORD"] = password
	minioSecret, err := m.infrastructureSecret("minio-password")
	if err != nil {
		return err
	}
	if err = m.SaveSecret("minio.env", []byte("MINIO_ROOT_USER=athena-instance\nMINIO_ROOT_PASSWORD="+minioSecret+"\nMINIO_REGION_NAME=us-east-1\n")); err != nil {
		return err
	}
	image := "athena-minio:9e49d5e7a648-go1.27.1"
	if _, err = m.Docker.Exec(ctx, "docker", "image", "inspect", image); err != nil {
		if _, err = m.Docker.Exec(ctx, "docker", "build", "--file", filepath.Join(m.Key.Checkout, "deploy/minio/Dockerfile.server"), "--tag", image, filepath.Join(m.Key.Checkout, "deploy/minio")); err != nil {
			return errors.New("MinIO image build failed")
		}
	}
	minio, err := m.ensureInfrastructureContainer(ctx, "minio", image, "/data", "9000", []string{"--env-file", filepath.Join(m.Key.Dir(), "minio.env")}, []string{"server", "/data", "--address", ":9000"})
	if err != nil {
		return err
	}
	address, err := m.containerAddress(ctx, minio, "9000/tcp")
	if err != nil {
		return err
	}
	endpoint := "http://" + address
	bucket := "athena-account-avatars"
	s3client := s3.New(s3.Options{Region: "us-east-1", Credentials: credentials.NewStaticCredentialsProvider("athena-instance", minioSecret, ""), BaseEndpoint: aws.String(endpoint), UsePathStyle: true})
	for {
		_, err = s3client.ListBuckets(ctx, &s3.ListBucketsInput{})
		if err == nil {
			break
		}
		select {
		case <-ctx.Done():
			return errors.New("instance MinIO readiness failed")
		case <-time.After(100 * time.Millisecond):
		}
	}
	if _, err = s3client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)}); err != nil {
		if _, err = s3client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)}); err != nil {
			return errors.New("instance avatar bucket initialization failed")
		}
	}
	env["ATHENA_ACCOUNT_AVATAR_S3_ENDPOINT"] = endpoint
	env["ATHENA_ACCOUNT_AVATAR_S3_REGION"] = "us-east-1"
	env["ATHENA_ACCOUNT_AVATAR_S3_BUCKET"] = bucket
	env["ATHENA_ACCOUNT_AVATAR_S3_ACCESS_KEY_ID"] = "athena-instance"
	env["ATHENA_ACCOUNT_AVATAR_S3_SECRET_ACCESS_KEY"] = minioSecret
	env["ATHENA_ACCOUNT_AVATAR_S3_PATH_STYLE"] = "true"
	return m.Update(func(s *State) error {
		if s.Endpoints == nil {
			s.Endpoints = map[string]string{}
		}
		s.Endpoints["redis"] = redisAddress
		s.Endpoints["minio"] = address
		return nil
	})
}
func (m *Manager) ensureInfrastructureContainer(ctx context.Context, component, image, mount, port string, options, args []string) (ResourceRef, error) {
	s, err := m.Status()
	if err != nil {
		return ResourceRef{}, err
	}
	if s.Phase != "starting" {
		return ResourceRef{}, errors.New("infrastructure requires starting instance")
	}
	if _, err = m.Docker.Exec(ctx, "docker", "image", "inspect", image); err != nil {
		if _, err = m.Docker.Exec(ctx, "docker", "pull", image); err != nil {
			return ResourceRef{}, errors.New("infrastructure image unavailable")
		}
	}
	name := "athena-" + m.Key.Namespace + "-" + component
	var volume, container ResourceRef
	for _, r := range s.Resources {
		if r.Name == name+"-data" {
			volume = r
		}
		if r.Name == name {
			container = r
		}
	}
	if volume.ID == "" {
		volume, err = m.CreateResource(ctx, "volume", name+"-data", nil)
		if err != nil {
			return container, err
		}
	} else if err = m.Docker.inspect(ctx, m.Key, volume); err != nil {
		return container, err
	}
	if container.ID == "" {
		options = append(options, "--publish", "127.0.0.1::"+port, "--mount", fmt.Sprintf("type=volume,source=%s,target=%s", volume.ID, mount))
		options = append(options, image)
		options = append(options, args...)
		container, err = m.CreateResource(ctx, "container", name, options)
		if err != nil {
			return container, err
		}
	} else if err = m.Docker.inspect(ctx, m.Key, container); err != nil {
		return container, err
	}
	_, err = m.Docker.Exec(ctx, "docker", "container", "start", container.ID)
	return container, err
}
