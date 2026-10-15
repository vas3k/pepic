package jobs

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vas3k/pepic/pepic/config"
)

func openTestQueue(t *testing.T, dir string, cfg config.JobsConfig, process Processor) *Queue {
	t.Helper()
	q, err := Open(dir, cfg, 1024, 3, process, func() (func(), error) { return func() {}, nil })
	if err != nil {
		t.Fatal(err)
	}
	return q
}

func waitJob(t *testing.T, q *Queue, id, state string) Job {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		j, err := q.Get(id)
		if err != nil {
			t.Fatal(err)
		}
		if j.State == state {
			return j
		}
		time.Sleep(10 * time.Millisecond)
	}
	j, _ := q.Get(id)
	t.Fatalf("expected %s, got %+v", state, j)
	return Job{}
}

func TestQueuedAndRunningJobsRecover(t *testing.T) {
	for _, running := range []bool{false, true} {
		t.Run(map[bool]string{false: "queued", true: "running"}[running], func(t *testing.T) {
			dir := t.TempDir()
			cfg := config.Defaults().Jobs
			entered := make(chan struct{})
			q := openTestQueue(t, dir, cfg, func(ctx context.Context, _ []byte) (Result, error) {
				close(entered)
				<-ctx.Done()
				return Result{}, ctx.Err()
			})
			job, err := q.Submit([]io.Reader{strings.NewReader("original")})
			if err != nil {
				t.Fatal(err)
			}
			if running {
				q.Start()
				select {
				case <-entered:
				case <-time.After(5 * time.Second):
					t.Fatal("worker did not start")
				}
			}
			if err = q.Close(); err != nil {
				t.Fatal(err)
			}
			var calls atomic.Int32
			q = openTestQueue(t, dir, cfg, func(_ context.Context, data []byte) (Result, error) {
				calls.Add(1)
				if string(data) != "original" {
					t.Error("lost staged bytes")
				}
				return Result{Filename: "result", URL: "https://media/result"}, nil
			})
			defer q.Close()
			q.Start()
			got := waitJob(t, q, job.ID, "succeeded")
			if len(got.Results) != 1 || calls.Load() != 1 {
				t.Fatalf("unexpected result %+v", got)
			}
			if _, err = os.Stat(filepath.Join(dir, job.ID, "0")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("private input was not removed")
			}
		})
	}
}

func TestQueueLimitsAndCleanup(t *testing.T) {
	cfg := config.Defaults().Jobs
	cfg.MaxPending = 1
	cfg.MaxBytes = 1024
	dir := t.TempDir()
	q := openTestQueue(t, dir, cfg, func(context.Context, []byte) (Result, error) { return Result{}, errors.New("private decoder details") })
	defer q.Close()
	if _, err := q.Submit([]io.Reader{bytes.NewReader(make([]byte, 1025))}); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("size guard: %v", err)
	}
	if q.bytes != 0 || q.reserved != 0 {
		t.Fatal("failed submission leaked capacity")
	}
	job, err := q.Submit([]io.Reader{strings.NewReader("bad")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = q.Submit([]io.Reader{strings.NewReader("other")}); !errors.Is(err, ErrFull) {
		t.Fatalf("queue guard: %v", err)
	}
	q.Start()
	got := waitJob(t, q, job.ID, "failed")
	if strings.Contains(got.Error, "private") {
		t.Fatal("internal error leaked")
	}
	q.mu.Lock()
	if q.bytes != 0 {
		t.Error("terminal input disk quota not released")
	}
	q.records[job.ID].Finished = time.Now().Add(-time.Duration(cfg.RetentionSeconds+1) * time.Second)
	q.cleanup()
	q.mu.Unlock()
	if _, err = q.Get(job.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal("expired job retained")
	}
	if _, err = q.Get("../../etc/passwd"); !errors.Is(err, ErrNotFound) {
		t.Fatal("path accepted")
	}
}

func TestSingleOwnerAndPanicRecovery(t *testing.T) {
	cfg := config.Defaults().Jobs
	dir := t.TempDir()
	q := openTestQueue(t, dir, cfg, func(context.Context, []byte) (Result, error) { panic("decoder panic") })
	defer q.Close()
	if other, err := Open(dir, cfg, 1024, 3, nil, nil); err == nil {
		other.Close()
		t.Fatal("second queue owner accepted")
	}
	job, err := q.Submit([]io.Reader{strings.NewReader("input")})
	if err != nil {
		t.Fatal(err)
	}
	q.Start()
	waitJob(t, q, job.ID, "failed")
}

func TestProcessingDeadlineAndBudget(t *testing.T) {
	cfg := config.Defaults().Jobs
	cfg.ProcessingTimeoutSeconds = 1
	q := openTestQueue(t, t.TempDir(), cfg, func(ctx context.Context, _ []byte) (Result, error) { <-ctx.Done(); return Result{}, ctx.Err() })
	defer q.Close()
	var allowed atomic.Bool
	q.acquire = func() (func(), error) {
		if !allowed.Load() {
			return nil, errors.New("busy")
		}
		return func() {}, nil
	}
	job, err := q.Submit([]io.Reader{strings.NewReader("input")})
	if err != nil {
		t.Fatal(err)
	}
	if q.runNext() {
		t.Fatal("shared processing budget ignored")
	}
	got, _ := q.Get(job.ID)
	if got.State != "queued" {
		t.Fatal("job consumed without processing capacity")
	}
	allowed.Store(true)
	q.Start()
	waitJob(t, q, job.ID, "failed")
}

func TestCrashRetryLimit(t *testing.T) {
	cfg := config.Defaults().Jobs
	dir := t.TempDir()
	q := openTestQueue(t, dir, cfg, func(context.Context, []byte) (Result, error) { t.Error("retry limit ignored"); return Result{}, nil })
	job, err := q.Submit([]io.Reader{strings.NewReader("input")})
	if err != nil {
		t.Fatal(err)
	}
	q.records[job.ID].Attempts = cfg.MaxAttempts
	q.Start()
	waitJob(t, q, job.ID, "failed")
	q.Close()
}
