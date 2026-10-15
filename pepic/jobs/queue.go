package jobs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/vas3k/pepic/pepic/config"
)

var ErrFull = errors.New("background upload queue is full")
var ErrTooLarge = errors.New("upload too large")
var ErrNotFound = errors.New("job not found")

type Result struct {
	MIME     string `json:"mime,omitempty"`
	Filename string `json:"filename"`
	URL      string `json:"url"`
}

type Job struct {
	ID          string    `json:"id"`
	SourceNames []string  `json:"source_names,omitempty"`
	State       string    `json:"state"`
	Files       int       `json:"files"`
	Results     []Result  `json:"results"`
	Error       string    `json:"error,omitempty"`
	Created     time.Time `json:"created"`
	Finished    time.Time `json:"finished,omitempty"`
	Attempts    int       `json:"attempts"`
	Bytes       int64     `json:"-"`
}

type Processor func(context.Context, []byte) (Result, error)
type Acquire func() (func(), error)

type Queue struct {
	mu       sync.Mutex
	root     *os.Root
	lock     *os.File
	cfg      config.JobsConfig
	limit    int64
	maxFiles int
	records  map[string]*Job
	reserved int
	bytes    int64
	process  Processor
	acquire  Acquire
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	wake     chan struct{}
}

func Open(dir string, cfg config.JobsConfig, limit int64, maxFiles int, process Processor, acquire Acquire) (*Queue, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	lock, err := root.OpenFile("lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		root.Close()
		return nil, err
	}
	if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		lock.Close()
		root.Close()
		return nil, fmt.Errorf("jobs directory is already in use: %w", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	q := &Queue{root: root, lock: lock, cfg: cfg, limit: limit, maxFiles: maxFiles, records: make(map[string]*Job), process: process, acquire: acquire, ctx: ctx, cancel: cancel, wake: make(chan struct{}, cfg.Workers)}
	if err = q.recover(); err != nil {
		q.Close()
		return nil, err
	}
	return q, nil
}

func validID(id string) bool {
	b, err := hex.DecodeString(id)
	return err == nil && len(b) == 32 && hex.EncodeToString(b) == id
}

func (q *Queue) persist(j *Job) error {
	data, err := json.Marshal(j)
	if err != nil {
		return err
	}
	if int64(len(data)) > q.cfg.MaxRecordBytes {
		return errors.New("job metadata too large")
	}
	name := filepath.Join(j.ID, "job.tmp")
	f, err := q.root.OpenFile(name, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err = q.root.Rename(name, filepath.Join(j.ID, "job.json")); err != nil {
		return err
	}
	dir, err := q.root.Open(j.ID)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func (q *Queue) recover() error {
	dir, err := q.root.Open(".")
	if err != nil {
		return err
	}
	defer dir.Close()
	entries, err := dir.ReadDir(-1)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() || !validID(entry.Name()) {
			continue
		}
		id := entry.Name()
		f, err := q.root.Open(filepath.Join(id, "job.json"))
		if errors.Is(err, os.ErrNotExist) {
			if err = q.root.RemoveAll(id); err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		var j Job
		err = json.NewDecoder(io.LimitReader(f, q.cfg.MaxRecordBytes)).Decode(&j)
		f.Close()
		if err != nil || j.ID != id || j.Files <= 0 || j.Files > q.maxFiles || len(j.Results) > j.Files {
			return fmt.Errorf("invalid persisted job %s", id)
		}
		switch j.State {
		case "processing", "queued":
			j.State = "queued"
		case "succeeded", "failed":
		default:
			return fmt.Errorf("invalid state for job %s", id)
		}
		q.records[id] = &j
		for i := 0; i < j.Files; i++ {
			info, err := q.root.Stat(filepath.Join(id, strconv.Itoa(i)))
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return err
			}
			j.Bytes += info.Size()
		}
		q.bytes += j.Bytes
		if j.State == "succeeded" || j.State == "failed" {
			q.removeInputs(&j)
		}
	}
	q.cleanup()
	if len(q.records) > q.cfg.MaxRecords {
		return fmt.Errorf("persisted jobs exceed jobs.max_records")
	}
	return nil
}

func (q *Queue) Submit(inputs []io.Reader, names ...string) (Job, error) {
	if len(names) != 0 && len(names) != len(inputs) {
		return Job{}, errors.New("file names do not match inputs")
	}
	if len(inputs) == 0 || len(inputs) > q.maxFiles {
		return Job{}, errors.New("invalid number of files")
	}
	q.mu.Lock()
	q.cleanup()
	pending := q.reserved
	for _, j := range q.records {
		if j.State == "queued" || j.State == "processing" {
			pending++
		}
	}
	if q.ctx.Err() != nil || pending >= q.cfg.MaxPending || len(q.records)+q.reserved >= q.cfg.MaxRecords || q.limit > q.cfg.MaxBytes-q.bytes {
		q.mu.Unlock()
		return Job{}, ErrFull
	}
	q.reserved++
	q.bytes += q.limit
	q.mu.Unlock()
	committed := false
	defer func() {
		if !committed {
			q.mu.Lock()
			q.reserved--
			q.bytes -= q.limit
			q.mu.Unlock()
		}
	}()
	var token [32]byte
	if _, err := rand.Read(token[:]); err != nil {
		return Job{}, err
	}
	j := Job{SourceNames: slices.Clone(names), ID: hex.EncodeToString(token[:]), State: "queued", Files: len(inputs), Results: []Result{}, Created: time.Now().UTC()}
	if err := q.root.Mkdir(j.ID, 0700); err != nil {
		return Job{}, err
	}
	defer func() {
		if !committed {
			q.root.RemoveAll(j.ID)
		}
	}()
	for i, input := range inputs {
		f, err := q.root.OpenFile(filepath.Join(j.ID, strconv.Itoa(i)), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return Job{}, err
		}
		n, err := io.Copy(f, io.LimitReader(input, q.limit-j.Bytes+1))
		j.Bytes += n
		if err == nil {
			err = f.Sync()
		}
		closeErr := f.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			return Job{}, err
		}
		if j.Bytes > q.limit {
			return Job{}, ErrTooLarge
		}
	}
	if err := q.persist(&j); err != nil {
		return Job{}, err
	}
	rootDir, err := q.root.Open(".")
	if err != nil {
		return Job{}, err
	}
	err = rootDir.Sync()
	rootDir.Close()
	if err != nil {
		return Job{}, err
	}
	q.mu.Lock()
	q.reserved--
	q.bytes += j.Bytes - q.limit
	q.records[j.ID] = &j
	committed = true
	result := clone(&j)
	q.mu.Unlock()
	select {
	case q.wake <- struct{}{}:
	default:
	}
	return result, nil
}

func clone(j *Job) Job {
	copy := *j
	copy.Results = slices.Clone(j.Results)
	copy.SourceNames = slices.Clone(j.SourceNames)
	return copy
}

func (q *Queue) Get(id string) (Job, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	j, ok := q.records[id]
	if !validID(id) || !ok {
		return Job{}, ErrNotFound
	}
	return clone(j), nil
}

func (q *Queue) Start() {
	for i := 0; i < q.cfg.Workers; i++ {
		q.wg.Add(1)
		go q.worker()
	}
}

func (q *Queue) Stop() { q.cancel() }

func (q *Queue) Close() error {
	q.cancel()
	q.wg.Wait()
	if q.lock != nil {
		q.lock.Close()
	}
	return q.root.Close()
}

func (q *Queue) worker() {
	defer q.wg.Done()
	ticker := time.NewTicker(time.Duration(q.cfg.PollSeconds) * time.Second)
	defer ticker.Stop()
	for {
		if q.ctx.Err() != nil {
			return
		}
		if q.runNext() {
			continue
		}
		select {
		case <-q.ctx.Done():
			return
		case <-ticker.C:
		case <-q.wake:
		}
	}
}

func (q *Queue) runNext() bool {
	q.mu.Lock()
	q.cleanup()
	var next *Job
	for _, j := range q.records {
		if j.State == "queued" && (next == nil || j.Created.Before(next.Created)) {
			next = j
		}
	}
	if next == nil {
		q.mu.Unlock()
		return false
	}
	release, err := q.acquire()
	if err != nil {
		q.mu.Unlock()
		return false
	}
	next.State = "processing"
	next.Attempts++
	if err = q.persist(next); err != nil {
		next.State = "queued"
		next.Attempts--
		q.mu.Unlock()
		release()
		log.Printf("Persist job: %v", err)
		return false
	}
	work := clone(next)
	q.mu.Unlock()
	defer release()
	var processErr error
	if work.Attempts > q.cfg.MaxAttempts {
		processErr = errors.New("job retry limit exceeded")
	}
	for i := len(work.Results); i < work.Files && processErr == nil; i++ {
		f, err := q.root.Open(filepath.Join(work.ID, strconv.Itoa(i)))
		if err != nil {
			processErr = err
			break
		}
		data, err := io.ReadAll(io.LimitReader(f, q.limit+1))
		f.Close()
		if err != nil {
			processErr = err
			break
		}
		if int64(len(data)) > q.limit {
			processErr = ErrTooLarge
			break
		}
		ctx, cancel := context.WithTimeout(q.ctx, time.Duration(q.cfg.ProcessingTimeoutSeconds)*time.Second)
		result, err := q.processSafely(ctx, data)
		cancel()
		if err != nil {
			processErr = err
			break
		}
		work.Results = append(work.Results, result)
		if err = q.persist(&work); err != nil {
			processErr = err
			break
		}
		q.mu.Lock()
		next.Results = slices.Clone(work.Results)
		q.mu.Unlock()
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.ctx.Err() != nil {
		next.State = "queued"
	} else {
		next.State = "succeeded"
		if processErr != nil {
			next.State = "failed"
			next.Error = "Media validation or processing failed"
			log.Printf("Job %s failed: %v", next.ID, processErr)
		}
		next.Finished = time.Now().UTC()
	}
	if err = q.persist(next); err != nil {
		log.Printf("Persist job result: %v", err)
		return false
	}
	if next.State != "queued" {
		q.removeInputs(next)
	}
	return true
}

func (q *Queue) removeInputs(j *Job) {
	for i := 0; i < j.Files; i++ {
		name := filepath.Join(j.ID, strconv.Itoa(i))
		info, err := q.root.Stat(name)
		if err != nil {
			continue
		}
		if err = q.root.Remove(name); err == nil {
			q.bytes -= info.Size()
			j.Bytes -= info.Size()
		}
	}
}

func (q *Queue) cleanup() {
	for id, j := range q.records {
		if j.State != "succeeded" && j.State != "failed" {
			continue
		}
		if j.Bytes > 0 {
			q.removeInputs(j)
		}
		if time.Since(j.Finished) < time.Duration(q.cfg.RetentionSeconds)*time.Second {
			continue
		}
		if err := q.root.RemoveAll(id); err != nil {
			log.Printf("Remove expired job: %v", err)
			continue
		}
		q.bytes -= j.Bytes
		delete(q.records, id)
	}
}

func (q *Queue) processSafely(ctx context.Context, data []byte) (result Result, err error) {
	defer func() {
		if value := recover(); value != nil {
			err = fmt.Errorf("processing panic: %v", value)
		}
	}()
	return q.process(ctx, data)
}

func (q *Queue) List() []Job {
	q.mu.Lock()
	result := make([]Job, 0, len(q.records))
	for _, job := range q.records {
		result = append(result, clone(job))
	}
	q.mu.Unlock()
	slices.SortFunc(result, func(a, b Job) int {
		activeA := a.State == "queued" || a.State == "processing"
		activeB := b.State == "queued" || b.State == "processing"
		if activeA != activeB {
			if activeA {
				return -1
			}
			return 1
		}
		if order := b.Created.Compare(a.Created); order != 0 {
			return order
		}
		return strings.Compare(a.ID, b.ID)
	})
	return result
}
