//go:build integration

package devruntime

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/useryege/athena/internal/accountavatar"
)

func TestAPIInfrastructureProvidesPrivateRedisAndAvatarStorage(t *testing.T) {
	m := databaseFixtureManager(t)
	ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
	defer cancel()
	env := map[string]string{}
	if e := m.prepareAPIInfrastructure(ctx, env); e != nil {
		t.Fatal(e)
	}
	client := redis.NewClient(&redis.Options{Addr: env["REDIS_SERVER"], Password: env["REDIS_PASSWORD"]})
	defer client.Close()
	if e := client.Set(ctx, "task9-marker", "value", 0).Err(); e != nil {
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
	initial, e := m.Status()
	if e != nil {
		t.Fatal(e)
	}
	t.Logf("isolated namespace=%s resources=%+v", m.Key.Namespace, initial.Resources)
	configPath := filepath.Join(m.Key.Dir(), "redis.conf")
	before, e := os.Stat(configPath)
	if e != nil {
		t.Fatal(e)
	}
	verifyPersistentAPIInfrastructure(t, ctx, env)
	for cycle := 1; cycle <= 2; cycle++ {
		if e = m.Stop(ctx); e != nil {
			t.Fatal(e)
		}
		if e = m.Update(func(s *State) error { s.Phase = "starting"; s.RunID = NewRunID(); return nil }); e != nil {
			t.Fatal(e)
		}
		env = map[string]string{}
		if e = m.prepareAPIInfrastructure(ctx, env); e != nil {
			after, statErr := os.Stat(configPath)
			if statErr == nil {
				t.Logf("config same inode=%t before=%v after=%v", os.SameFile(before, after), before.Sys(), after.Sys())
			}
			for _, resource := range initial.Resources {
				if resource.Kind == "container" && strings.HasSuffix(resource.Name, "-redis") {
					if output, inspectErr := m.Docker.Exec(ctx, "docker", "container", "inspect", "--format", "{{json .State}}", resource.ID); inspectErr == nil {
						t.Logf("Redis state: %s", output)
					}
				}
			}
			t.Fatalf("persistent restart cycle %d: %v", cycle, e)
		}
		after, err := os.Stat(configPath)
		if err != nil {
			t.Fatal(err)
		}
		if !os.SameFile(before, after) {
			t.Fatal("restart replaced Redis config inode")
		}
		state, err := m.Status()
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(initial.Resources, state.Resources) {
			t.Fatalf("restart replaced resources: %+v", state.Resources)
		}
		verifyPersistentAPIInfrastructure(t, ctx, env)
		t.Logf("persistent restart cycle %d passed: same resources and config inode; Redis=%s MinIO=%s", cycle, env["REDIS_SERVER"], env["ATHENA_ACCOUNT_AVATAR_S3_ENDPOINT"])
	}
}

// verifyPersistentAPIInfrastructure checks fresh clients against the newly
// discovered private ports; Docker may allocate different ports after restart.
func verifyPersistentAPIInfrastructure(t *testing.T, ctx context.Context, env map[string]string) {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: env["REDIS_SERVER"], Password: env["REDIS_PASSWORD"]})
	defer client.Close()
	if value, err := client.Get(ctx, "task9-marker").Result(); err != nil || value != "value" {
		t.Fatalf("persistent Redis marker = %q: %v", value, err)
	}
	anonymous := redis.NewClient(&redis.Options{Addr: env["REDIS_SERVER"]})
	defer anonymous.Close()
	if err := anonymous.Get(ctx, "task9-marker").Err(); err == nil || !strings.Contains(err.Error(), "NOAUTH") {
		t.Fatalf("Redis must still require authentication: %v", err)
	}
	store, err := accountavatar.NewStore(ctx, accountavatar.Config{Endpoint: env["ATHENA_ACCOUNT_AVATAR_S3_ENDPOINT"], Region: env["ATHENA_ACCOUNT_AVATAR_S3_REGION"], Bucket: env["ATHENA_ACCOUNT_AVATAR_S3_BUCKET"], AccessKey: env["ATHENA_ACCOUNT_AVATAR_S3_ACCESS_KEY_ID"], SecretKey: env["ATHENA_ACCOUNT_AVATAR_S3_SECRET_ACCESS_KEY"], UsePathStyle: true})
	if err != nil {
		t.Fatal(err)
	}
	object, err := store.Get(ctx, "task9-marker")
	if err != nil {
		t.Fatal(err)
	}
	defer object.Body.Close()
	body, err := io.ReadAll(object.Body)
	if err != nil || string(body) != "fixture" || object.ContentType != "image/png" {
		t.Fatalf("persistent avatar = %q type=%q: %v", body, object.ContentType, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, env["ATHENA_ACCOUNT_AVATAR_S3_ENDPOINT"]+"/"+env["ATHENA_ACCOUNT_AVATAR_S3_BUCKET"]+"/task9-marker", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("anonymous avatar status = %d", response.StatusCode)
	}
}
