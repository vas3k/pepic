package jobs

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/vas3k/pepic/pepic/config"
)

func TestListIsIndependentSnapshot(t *testing.T) {
	q := openTestQueue(t, t.TempDir(), config.Defaults().Jobs, func(context.Context, []byte) (Result, error) { return Result{}, nil })
	defer q.Close()
	job, err := q.Submit([]io.Reader{strings.NewReader("data")}, "photo.png")
	if err != nil {
		t.Fatal(err)
	}
	list := q.List()
	if len(list) != 1 || list[0].ID != job.ID || list[0].SourceNames[0] != "photo.png" {
		t.Fatalf("bad snapshot: %+v", list)
	}
	list[0].SourceNames[0] = "changed"
	list[0].State = "failed"
	current, err := q.Get(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.State != "queued" || current.SourceNames[0] != "photo.png" {
		t.Fatal("listing mutated persisted job")
	}
}
