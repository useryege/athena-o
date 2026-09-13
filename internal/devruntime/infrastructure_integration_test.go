//go:build integration

package devruntime

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/useryege/athena/internal/accountavatar"
	"strings"
	"testing"
	"time"
)

func TestAPIInfrastructureProvidesPrivateRedisAndAvatarStorage(t *testing.T) {
	m := databaseFixtureManager(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	env := map[string]string{}
	if e := m.prepareAPIInfrastructure(ctx, env); e != nil {
		t.Fatal(e)
	}
	client := redis.NewClient(&redis.Options{Addr: env["REDIS_SERVER"], Password: env["REDIS_PASSWORD"]})
	defer client.Close()
	if e := client.Set(ctx, "task9-marker", "value", time.Minute).Err(); e != nil {
		t.Fatal(e)
	}
	cfg := accountavatar.Config{Endpoint: env["ATHENA_ACCOUNT_AVATAR_S3_ENDPOINT"], Region: env["ATHENA_ACCOUNT_AVATAR_S3_REGION"], Bucket: env["ATHENA_ACCOUNT_AVATAR_S3_BUCKET"], AccessKey: env["ATHENA_ACCOUNT_AVATAR_S3_ACCESS_KEY_ID"], SecretKey: env["ATHENA_ACCOUNT_AVATAR_S3_SECRET_ACCESS_KEY"], UsePathStyle: true}
	store, e := accountavatar.NewStore(ctx, cfg)
	if e != nil {
		t.Fatal(e)
	}
	_, e = store.Put(ctx, "task9-marker", strings.NewReader("fixture"), 7, "image/png")
	if e != nil {
		t.Fatal(e)
	}
}
