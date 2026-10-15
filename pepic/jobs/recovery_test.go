package jobs

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/vas3k/pepic/pepic/config"
)

func TestRecoveryKeepsCompletedBatchResults(t *testing.T) {
	cfg := config.Defaults().Jobs
	dir := t.TempDir()
	entered := make(chan struct{})
	q := openTestQueue(t, dir, cfg, func(ctx context.Context, data []byte) (Result, error) {
		if string(data) == "first" {
			return Result{Filename: "first"}, nil
		}
		close(entered)
		<-ctx.Done()
		return Result{}, ctx.Err()
	})
	job, err := q.Submit([]io.Reader{strings.NewReader("first"), strings.NewReader("second")})
	if err != nil {
		t.Fatal(err)
	}
	q.Start()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("second file did not start")
	}
	q.Close()
	q = openTestQueue(t, dir, cfg, func(_ context.Context, data []byte) (Result, error) {
		if string(data) != "second" {
			t.Error("completed file processed twice")
		}
		return Result{Filename: string(data)}, nil
	})
	defer q.Close()
	q.Start()
	done := waitJob(t, q, job.ID, "succeeded")
	if len(done.Results) != 2 || done.Results[0].Filename != "first" || done.Results[1].Filename != "second" {
		t.Fatalf("results lost: %+v", done.Results)
	}
}
