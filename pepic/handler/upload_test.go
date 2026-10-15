package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/vas3k/pepic/pepic/config"
	"github.com/vas3k/pepic/pepic/storage"
)

func setupUpload(t *testing.T) (*echo.Echo, string) {
	t.Helper()
	old := config.App
	t.Cleanup(func() { config.App = old })
	config.App.Global.MaxUploadSize = "1M"
	config.App.Global.SecretCode = ""
	config.App.Global.BaseUrl = "http://localhost/"
	config.App.Global.FileTreeSplitChars = 3
	config.App.Images.StoreOriginals = true
	config.App.Videos.StoreOriginals = true
	dir := t.TempDir()
	h := &PepicHandler{Storage: storage.NewStorage(storage.NewFileSystemBackend(dir))}
	e := echo.New()
	e.HTTPErrorHandler = h.ErrorHandler
	e.POST("/upload/bytes/", h.UploadBodyBytes)
	e.POST("/upload/multipart/", h.UploadMultipart)
	e.GET("/:name", h.GetOriginalFile)
	return e, dir
}

func pngBytes(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func request(e *echo.Echo, route string, body io.Reader, contentType string, chunked bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, route, body)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", contentType)
	if chunked {
		req.ContentLength = -1
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func storedFiles(t *testing.T, dir string) int {
	t.Helper()
	n := 0
	if err := filepath.WalkDir(dir, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			n++
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestRawUploadValidation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		data   []byte
		status int
	}{
		{"png", pngBytes(t), 201},
		{"zip", []byte("PK\x03\x04evil archive"), 415},
		{"exe", []byte("MZ executable"), 415},
		{"msi", []byte("\xd0\xcf\x11\xe0\xa1\xb1\x1a\xe1installer"), 415},
		{"svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`), 415},
		{"empty", nil, 415},
		{"fake PNG", append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 100)...), 415},
		{"truncated PNG", pngBytes(t)[:33], 415},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, dir := setupUpload(t)
			rec := request(e, "/upload/bytes/", bytes.NewReader(tc.data), "image/png", false)
			if rec.Code != tc.status {
				t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
			}
			if tc.status != 201 && storedFiles(t, dir) != 0 {
				t.Fatal("rejected data was stored")
			}
			if tc.status == 201 && storedFiles(t, dir) != 1 {
				t.Fatal("valid media not stored")
			}
			if tc.status == 415 && !strings.Contains(rec.Body.String(), `"error"`) {
				t.Fatal("missing JSON error")
			}
		})
	}
}

func TestMultipartSpoofingAndCleanup(t *testing.T) {
	e, dir := setupUpload(t)
	for _, name := range []string{"evil.jpg", "evil.mp4", "evil.zip"} {
		var body bytes.Buffer
		w := multipart.NewWriter(&body)
		p, _ := w.CreateFormFile("media", name)
		p.Write([]byte("PK\x03\x04not media"))
		w.Close()
		rec := request(e, "/upload/multipart/", &body, w.FormDataContentType(), false)
		if rec.Code != 415 {
			t.Fatalf("%s: %d %s", name, rec.Code, rec.Body.String())
		}
	}
	if storedFiles(t, dir) != 0 {
		t.Fatal("spoofed files stored")
	}
}

func TestUploadLimits(t *testing.T) {
	for _, chunked := range []bool{false, true} {
		for _, route := range []string{"/upload/bytes/", "/upload/multipart/"} {
			t.Run(route+map[bool]string{true: "chunked", false: "known"}[chunked], func(t *testing.T) {
				e, dir := setupUpload(t)
				config.App.Global.MaxUploadSize = "1K"
				var body bytes.Buffer
				ct := "image/png"
				if strings.Contains(route, "multipart") {
					w := multipart.NewWriter(&body)
					p, _ := w.CreateFormFile("media", "big.png")
					p.Write(make([]byte, 2000))
					w.Close()
					ct = w.FormDataContentType()
				} else {
					body.Write(make([]byte, 2000))
				}
				rec := request(e, route, &body, ct, chunked)
				if rec.Code != 413 {
					t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
				}
				if storedFiles(t, dir) != 0 {
					t.Fatal("oversized data stored")
				}
			})
		}
	}
}

type failingReader struct{ data []byte }

func (r *failingReader) Read(p []byte) (int, error) {
	if r.data != nil {
		n := copy(p, r.data)
		r.data = nil
		return n, nil
	}
	return 0, errors.New("broken body")
}

func TestReadFailureDoesNotStorePrefix(t *testing.T) {
	e, dir := setupUpload(t)
	rec := request(e, "/upload/bytes/", &failingReader{pngBytes(t)}, "image/png", true)
	if rec.Code != 400 || storedFiles(t, dir) != 0 {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}

func TestProcessingBusy(t *testing.T) {
	e, dir := setupUpload(t)
	a, _ := acquireProcessing()
	defer a()
	b, _ := acquireProcessing()
	defer b()
	rec := request(e, "/upload/bytes/", bytes.NewReader(pngBytes(t)), "image/png", false)
	if rec.Code != 503 || storedFiles(t, dir) != 0 {
		t.Fatalf("%d", rec.Code)
	}
}

func TestAuthRequired(t *testing.T) {
	e, dir := setupUpload(t)
	config.App.Global.SecretCode = "secret"
	rec := request(e, "/upload/bytes/", bytes.NewReader(pngBytes(t)), "image/png", false)
	if rec.Code != 401 || storedFiles(t, dir) != 0 {
		t.Fatalf("%d", rec.Code)
	}
}

func TestDownloadGuard(t *testing.T) {
	e, _ := setupUpload(t)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest("GET", "/"+strings.Repeat("a", 64)+".exe", nil))
	if rec.Code != 404 {
		t.Fatalf("%d", rec.Code)
	}
}

func TestMultipartValidAndFileCount(t *testing.T) {
	for _, count := range []int{1, 11} {
		e, dir := setupUpload(t)
		var body bytes.Buffer
		w := multipart.NewWriter(&body)
		for i := 0; i < count; i++ {
			p, _ := w.CreateFormFile("media", "photo.png")
			p.Write(pngBytes(t))
		}
		w.Close()
		rec := request(e, "/upload/multipart/", &body, w.FormDataContentType(), false)
		want := 201
		if count > 10 {
			want = 400
		}
		if rec.Code != want {
			t.Fatalf("%d files: %d %s", count, rec.Code, rec.Body.String())
		}
		if count > 10 && storedFiles(t, dir) != 0 {
			t.Fatal("over-limit batch stored")
		}
	}
}

func TestDownloadSecurityHeaders(t *testing.T) {
	e, dir := setupUpload(t)
	rec := request(e, "/upload/bytes/", bytes.NewReader(pngBytes(t)), "image/png", false)
	if rec.Code != 201 {
		t.Fatal(rec.Body.String())
	}
	var name string
	filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			rel, _ := filepath.Rel(filepath.Join(dir, "orig"), p)
			name = strings.ReplaceAll(rel, string(filepath.Separator), "")
		}
		return err
	})
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest("GET", "/"+name, nil))
	if rec.Code != 200 || rec.Header().Get("X-Content-Type-Options") != "nosniff" || rec.Header().Get("Content-Security-Policy") == "" {
		t.Fatalf("%d %v", rec.Code, rec.Header())
	}
}

func TestConfiguredFormatControlsUploadAndDownload(t *testing.T) {
	e, _ := setupUpload(t)
	original := config.App.Media.Formats
	formats := make(map[string]config.MediaFormat, len(original))
	for mimeType, format := range original {
		formats[mimeType] = format
	}
	config.App.Media.Formats = formats

	uploaded := request(e, "/upload/bytes/", bytes.NewReader(pngBytes(t)), "image/png", false)
	if uploaded.Code != http.StatusCreated {
		t.Fatal(uploaded.Body.String())
	}
	var response struct {
		Uploaded []string `json:"uploaded"`
	}
	if err := json.Unmarshal(uploaded.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}

	format := formats["image/png"]
	delete(formats, "image/png")
	rejected := request(e, "/upload/bytes/", bytes.NewReader(pngBytes(t)), "image/png", false)
	if rejected.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("disabled format: %d", rejected.Code)
	}
	download := httptest.NewRecorder()
	e.ServeHTTP(download, httptest.NewRequest("GET", response.Uploaded[0], nil))
	if download.Code != http.StatusNotFound {
		t.Fatalf("disabled download: %d", download.Code)
	}

	format.Extension = ".picture"
	format.Aliases = []string{"image/png"}
	formats["image/x-test-picture"] = format
	uploaded = request(e, "/upload/bytes/", bytes.NewReader(pngBytes(t)), "image/png", false)
	if uploaded.Code != http.StatusCreated {
		t.Fatal(uploaded.Body.String())
	}
	if err := json.Unmarshal(uploaded.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(response.Uploaded[0], ".picture") {
		t.Fatal(response.Uploaded)
	}
	download = httptest.NewRecorder()
	e.ServeHTTP(download, httptest.NewRequest("GET", response.Uploaded[0], nil))
	if download.Code != http.StatusOK || download.Header().Get("Content-Type") != "image/x-test-picture" {
		t.Fatalf("%d %v", download.Code, download.Header())
	}
}

func TestConfiguredWorkerLimit(t *testing.T) {
	e, _ := setupUpload(t)
	config.App.Safety.MaxConcurrentJobs = 1
	release, err := acquireProcessing()
	if err != nil {
		t.Fatal(err)
	}
	response := request(e, "/upload/bytes/", bytes.NewReader(pngBytes(t)), "image/png", false)
	release()
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("busy: %d", response.Code)
	}
	response = request(e, "/upload/bytes/", bytes.NewReader(pngBytes(t)), "image/png", false)
	if response.Code != http.StatusCreated {
		t.Fatalf("released: %d", response.Code)
	}
}

func TestConfiguredPixelLimit(t *testing.T) {
	e, _ := setupUpload(t)
	config.App.Safety.MaxImagePixels = 1
	response := request(e, "/upload/bytes/", bytes.NewReader(pngBytes(t)), "image/png", false)
	if response.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("pixel limit: %d", response.Code)
	}
}

func TestEchoBodyLimitOnChunkedUploads(t *testing.T) {
	for _, route := range []string{"/upload/bytes/", "/upload/multipart/"} {
		t.Run(route, func(t *testing.T) {
			e, dir := setupUpload(t)
			config.App.Global.MaxUploadSize = "1K"
			e.Use(middleware.BodyLimit(1000))
			var body bytes.Buffer
			contentType := "application/octet-stream"
			if strings.Contains(route, "multipart") {
				writer := multipart.NewWriter(&body)
				part, err := writer.CreateFormFile("media", "large.png")
				if err != nil {
					t.Fatal(err)
				}
				part.Write(make([]byte, 2000))
				writer.Close()
				contentType = writer.FormDataContentType()
			} else {
				body.Write(make([]byte, 2000))
			}
			response := request(e, route, &body, contentType, true)
			if response.Code != http.StatusRequestEntityTooLarge || storedFiles(t, dir) != 0 {
				t.Fatalf("%d %s", response.Code, response.Body.String())
			}
		})
	}
}
