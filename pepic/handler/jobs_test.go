package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/vas3k/pepic/pepic/config"
	"github.com/vas3k/pepic/pepic/jobs"
	"github.com/vas3k/pepic/pepic/processing"
	"github.com/vas3k/pepic/pepic/storage"
)

func setupJobs(t *testing.T) (*echo.Echo, *PepicHandler, string) {
	t.Helper()
	e, dir := setupUpload(t)
	config.App.Jobs.Dir = t.TempDir()
	h := &PepicHandler{Storage: storage.NewStorage(storage.NewFileSystemBackend(dir)), Processing: processing.Processing{Image: processing.NewImageBackend(), Video: processing.NewVideoBackend()}}
	if err := h.OpenJobs(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { h.Jobs.Close() })
	e.POST("/upload/jobs/bytes/", h.UploadJobBytes)
	e.POST("/upload/jobs/multipart/", h.UploadJobMultipart)
	e.GET("/jobs/", h.ListJobs)
	e.GET("/jobs/:id", h.GetJob)
	return e, h, dir
}

func jobRequest(t *testing.T, e *echo.Echo, route string, body io.Reader, contentType string) jobResponse {
	t.Helper()
	rec := request(e, route, body, contentType, false)
	if rec.Code != 202 {
		t.Fatalf("async upload: %d %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("upload must not be cached")
	}
	var job jobResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &job); err != nil {
		t.Fatal(err)
	}
	if rec.Header().Get("Location") != job.StatusURL || job.ID == "" {
		t.Fatal("missing job location")
	}
	return job
}

func awaitJob(t *testing.T, e *echo.Echo, url string) jobResponse {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		req := httptest.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Accept", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Fatalf("poll: %d %s", rec.Code, rec.Body.String())
		}
		if rec.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("job status must not be cached")
		}
		var job jobResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &job); err != nil {
			t.Fatal(err)
		}
		if job.State == "succeeded" || job.State == "failed" {
			return job
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("job did not finish")
	return jobResponse{}
}

func TestBackgroundUploadAndExistingAPI(t *testing.T) {
	e, h, dir := setupJobs(t)
	entered := make(chan struct{})
	unblock := make(chan struct{})
	h.Jobs.Close()
	q, err := jobs.Open(config.App.Jobs.Dir, config.App.Jobs, 1000000, 10, func(ctx context.Context, data []byte) (jobs.Result, error) {
		close(entered)
		select {
		case <-unblock:
		case <-ctx.Done():
			return jobs.Result{}, ctx.Err()
		}
		file, err := h.uploadBytesContext(ctx, "", data)
		if err != nil {
			return jobs.Result{}, err
		}
		return jobs.Result{Filename: file.Filename, URL: file.Url()}, nil
	}, acquireProcessing)
	if err != nil {
		t.Fatal(err)
	}
	h.Jobs = q
	h.Jobs.Start()
	job := jobRequest(t, e, "/upload/jobs/bytes/", bytes.NewReader(pngBytes(t)), "image/png")
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("job did not start")
	}
	if storedFiles(t, dir) != 0 {
		t.Fatal("unvalidated job was published")
	}
	legacy := request(e, "/upload/bytes/", bytes.NewReader(pngBytes(t)), "image/png", false)
	if legacy.Code != 201 || !strings.Contains(legacy.Body.String(), `"uploaded"`) {
		t.Fatalf("legacy API changed: %d %s", legacy.Code, legacy.Body.String())
	}
	close(unblock)
	done := awaitJob(t, e, job.StatusURL)
	if done.State != "succeeded" || len(done.Results) != 1 || !strings.Contains(legacy.Body.String(), done.Results[0].URL) {
		t.Fatalf("async/legacy output mismatch: %+v", done)
	}
}

func TestBackgroundMultipartValidation(t *testing.T) {
	for _, valid := range []bool{true, false} {
		t.Run(map[bool]string{true: "image", false: "disguised archive"}[valid], func(t *testing.T) {
			e, h, dir := setupJobs(t)
			h.Jobs.Start()
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			part, err := writer.CreateFormFile("media", "spoof.jpg")
			if err != nil {
				t.Fatal(err)
			}
			if valid {
				part.Write(pngBytes(t))
			} else {
				part.Write([]byte("PK\x03\x04archive"))
			}
			writer.Close()
			job := jobRequest(t, e, "/upload/jobs/multipart/", &body, writer.FormDataContentType())
			done := awaitJob(t, e, job.StatusURL)
			if valid && (done.State != "succeeded" || storedFiles(t, dir) != 1) {
				t.Fatalf("valid upload failed: %+v", done)
			}
			if !valid && (done.State != "failed" || storedFiles(t, dir) != 0) {
				t.Fatalf("unsafe upload published: %+v", done)
			}
		})
	}
}

func TestBackgroundVideo(t *testing.T) {
	e, h, _ := setupJobs(t)
	h.Jobs.Start()
	config.App.Videos.StoreOriginals = false
	filename := filepath.Join(t.TempDir(), "clip.webm")
	if out, err := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "color=c=red:s=32x32:r=5", "-t", "0.4", filename).CombinedOutput(); err != nil {
		t.Fatalf("fixture: %v %s", err, out)
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	job := jobRequest(t, e, "/upload/jobs/bytes/", bytes.NewReader(data), "application/octet-stream")
	done := awaitJob(t, e, job.StatusURL)
	if done.State != "succeeded" || len(done.Results) != 1 || !strings.HasSuffix(done.Results[0].Filename, ".mp4") || done.Results[0].MIME != "video/mp4" {
		t.Fatalf("video failed: %+v", done)
	}
}

func TestBackgroundAuthAndLimits(t *testing.T) {
	e, h, _ := setupJobs(t)
	config.App.Global.SecretCode = "secret"
	rec := request(e, "/upload/jobs/bytes/", bytes.NewReader(pngBytes(t)), "image/png", false)
	if rec.Code != 401 {
		t.Fatalf("unauthenticated upload: %d", rec.Code)
	}
	job := jobRequest(t, e, "/upload/jobs/bytes/?code=secret", bytes.NewReader(pngBytes(t)), "image/png")
	req := httptest.NewRequest(http.MethodGet, job.StatusURL, nil)
	req.Header.Set("Accept", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatalf("unauthenticated poll: %d", rec.Code)
	}
	config.App.Global.SecretCode = ""
	rec = request(e, "/upload/jobs/bytes/", bytes.NewReader(make([]byte, 1000001)), "image/png", true)
	if rec.Code != 413 {
		t.Fatalf("size guard: %d %s", rec.Code, rec.Body.String())
	}
	h.Jobs.Start()
	done := awaitJob(t, e, job.StatusURL)
	if done.State != "succeeded" {
		t.Fatal("authenticated upload failed")
	}
}
