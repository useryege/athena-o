//go:build linux

package devruntime

import (
	"bufio"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestPidfdWaitsForThreadGroupExitAfterLeaderMetadataDisappears(t *testing.T) {
	compiler, err := exec.LookPath("gcc")
	if err != nil {
		t.Skip("thread-exit fixture requires gcc")
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "exit.c")
	binary := filepath.Join(dir, "thread-exit")
	code := `#include <pthread.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <stdatomic.h>
static atomic_int ready;
static int delay;
static void *worker(void *unused) {
 while (!atomic_load(&ready)) usleep(1000);
 usleep(delay*1000);
 return NULL;
}
int main(int argc,char **argv) {
 delay=atoi(argv[1]);
 pthread_t thread; pthread_create(&thread,NULL,worker,NULL);
 puts("ready"); fflush(stdout);
 getchar(); atomic_store(&ready,1); pthread_exit(NULL);
}`
	if err = os.WriteFile(source, []byte(code), 0600); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(compiler, "-pthread", "-o", binary, source).CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	for _, scenario := range []struct {
		name    string
		delay   int
		signal  bool
		timeout time.Duration
	}{
		{"wait-for-exit", 150, false, time.Second},
		{"signal-observes-exit", 150, true, time.Second},
		{"reject-live-group", 2000, false, time.Second},
		{"do-not-signal-live-group", 2000, true, time.Second},
		{"honor-short-context", 150, false, 20 * time.Millisecond},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			delay := scenario.delay
			cmd := exec.Command(binary, strconv.Itoa(delay))
			cmd.Env = append(os.Environ(), RunIDEnv+"="+NewRunID())
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			input, err := cmd.StdinPipe()
			if err != nil {
				t.Fatal(err)
			}
			output, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err = cmd.Start(); err != nil {
				t.Fatal(err)
			}
			defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
			if _, err = bufio.NewReader(output).ReadString('\n'); err != nil {
				t.Fatal(err)
			}
			identity, err := ReadProcess(cmd.Process.Pid)
			if err != nil {
				t.Fatal(err)
			}
			fd, err := unix.PidfdOpen(identity.PID, 0)
			if err != nil {
				t.Fatal(err)
			}
			defer unix.Close(fd)
			if _, err = input.Write([]byte("x")); err != nil {
				t.Fatal(err)
			}
			input.Close()
			deadline := time.Now().Add(time.Second)
			for {
				_, err = ReadProcess(identity.PID)
				if errors.Is(err, os.ErrNotExist) {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("leader metadata did not disappear")
				}
				time.Sleep(time.Millisecond)
			}
			if exited, err := pidfdExited(fd, 0); err != nil || exited {
				t.Fatal("fixture did not hold worker after leader exit", err)
			}
			start := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), scenario.timeout)
			defer cancel()
			if scenario.signal {
				err = SignalProcess(identity, syscall.SIGTERM)
			} else {
				err = WaitProcess(ctx, identity)
			}
			if scenario.timeout < time.Second {
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("expected caller deadline while group remains live: %v", err)
				}
				if time.Since(start) > 100*time.Millisecond {
					t.Fatal("identity verification exceeded caller deadline")
				}
				if exited, _ := pidfdExited(fd, 0); exited {
					t.Fatal("deadline case did not retain live worker")
				}
				return
			}
			if delay == 150 {
				if err != nil {
					t.Fatalf("cannot wait for positive thread-group exit: %v", err)
				}
				if exited, _ := pidfdExited(fd, 0); !exited {
					t.Fatal("accepted missing identity without kernel exit")
				}
				if err = cmd.Wait(); err != nil {
					t.Fatal("verification signalled an unverified worker", err)
				}
			} else {
				if err == nil {
					t.Fatal("accepted a still-live group with unavailable identity")
				}
				if time.Since(start) > time.Second {
					t.Fatal("unavailable identity exceeded finite verification budget")
				}
				if exited, _ := pidfdExited(fd, 0); exited {
					t.Fatal("verification signalled the unverified live group")
				}
			}
		})
	}
}
