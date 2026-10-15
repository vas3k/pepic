package handler

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vas3k/pepic/pepic/config"
	"github.com/vas3k/pepic/pepic/processing"
	"github.com/vas3k/pepic/pepic/storage"
	"github.com/vas3k/pepic/pepic/template"
)

func TestMediaRoutesCompatibility(t *testing.T) {
	e, dir := setupUpload(t)
	h := &PepicHandler{Storage: storage.NewStorage(storage.NewFileSystemBackend(dir)), Processing: processing.Processing{Image: processing.NewImageBackend(), Video: processing.NewVideoBackend()}}
	e.GET("/:length/:name", h.GetResizedFile)
	e.GET("/meta/:name", h.GetMeta)
	e.GET("/", h.Index)
	e.Renderer = template.NewTemplateRenderer("../../html")
	var data bytes.Buffer
	if err := jpeg.Encode(&data, image.NewRGBA(image.Rect(0, 0, 640, 320)), nil); err != nil {
		t.Fatal(err)
	}
	filename := fmt.Sprintf("%x.jpg", sha256.Sum256(data.Bytes()))
	rec := request(e, "/upload/bytes/", bytes.NewReader(data.Bytes()), "application/octet-stream", false)
	if rec.Code != 201 || !strings.Contains(rec.Body.String(), filename) {
		t.Fatalf("upload: %d %s", rec.Code, rec.Body.String())
	}
	get := func(route string, headers map[string]string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, route, nil)
		req.Header.Set("Accept", "application/json")
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		out := httptest.NewRecorder()
		e.ServeHTTP(out, req)
		return out
	}
	for _, route := range []string{"/" + filename, "/full/" + filename} {
		out := get(route, nil)
		if out.Code != 200 || !bytes.Equal(out.Body.Bytes(), data.Bytes()) {
			t.Fatalf("original %s: %d", route, out.Code)
		}
	}
	out := get("/"+filename, map[string]string{"Range": "bytes=0-9"})
	if out.Code != 206 || !bytes.Equal(out.Body.Bytes(), data.Bytes()[:10]) {
		t.Fatalf("range: %d", out.Code)
	}
	config.App.Images.LiveResize = false
	out = get("/300/"+filename, nil)
	if out.Code != 200 || !bytes.Equal(out.Body.Bytes(), data.Bytes()) {
		t.Fatalf("disabled resize: %d", out.Code)
	}
	config.App.Images.LiveResize = true
	out = get("/300/"+filename, nil)
	if out.Code != 200 {
		t.Fatalf("resize: %d %s", out.Code, out.Body.String())
	}
	dimensions, err := jpeg.DecodeConfig(bytes.NewReader(out.Body.Bytes()))
	if err != nil || dimensions.Width != 300 || dimensions.Height != 150 {
		t.Fatalf("dimensions: %+v %v", dimensions, err)
	}
	processingBudget.Lock()
	processingBudget.active = config.App.Safety.MaxConcurrentJobs
	processingBudget.Unlock()
	cached := get("/300/"+filename, nil)
	processingBudget.Lock()
	processingBudget.active = 0
	processingBudget.Unlock()
	if cached.Code != 200 || !bytes.Equal(cached.Body.Bytes(), out.Body.Bytes()) {
		t.Fatalf("cached resize under load: %d", cached.Code)
	}
	for _, route := range []string{"/", "/meta/" + filename} {
		out = get(route, nil)
		if out.Code != 200 || !strings.Contains(out.Header().Get("Content-Type"), "text/html") {
			t.Fatalf("page %s: %d %s", route, out.Code, out.Body.String())
		}
	}
	for _, route := range []string{"/300/" + strings.Repeat("0", 64) + ".jpg", "/300/not-a-hash.jpg"} {
		if out = get(route, nil); out.Code != 404 {
			t.Fatalf("missing %s: %d", route, out.Code)
		}
	}
}

func TestUploadConversionAndBrowserRedirect(t *testing.T) {
	e, dir := setupUpload(t)
	h := &PepicHandler{Storage: storage.NewStorage(storage.NewFileSystemBackend(dir)), Processing: processing.Processing{Image: processing.NewImageBackend(), Video: processing.NewVideoBackend()}}
	e.POST("/processed/", h.UploadBodyBytes)
	config.App.Images.StoreOriginals = false
	config.App.Images.AutoConvert = "image/jpeg"
	var data bytes.Buffer
	if err := png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 4, 2))); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/processed/", bytes.NewReader(data.Bytes()))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	filename := fmt.Sprintf("%x.jpg", sha256.Sum256(data.Bytes()))
	if rec.Code != 302 || rec.Header().Get("Location") != "/meta/"+filename {
		t.Fatalf("redirect: %d %s", rec.Code, rec.Header().Get("Location"))
	}
	file, err := h.Storage.GetFile("orig", filename)
	if err != nil {
		t.Fatal(err)
	}
	if err = h.Storage.ReadFileBytes(file, "orig"); err != nil {
		t.Fatal(err)
	}
	if _, err = jpeg.Decode(bytes.NewReader(file.Bytes)); err != nil {
		t.Fatal(err)
	}
}

func TestSecretCookieCompatibility(t *testing.T) {
	e, _ := setupUpload(t)
	config.App.Global.SecretCode = "test-secret"
	config.App.Global.BaseUrl = "https://media.example/"
	rec := request(e, "/upload/bytes/?code=test-secret", bytes.NewReader(pngBytes(t)), "application/octet-stream", false)
	if rec.Code != 201 {
		t.Fatalf("authenticated upload: %d", rec.Code)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].Secure || !cookies[0].HttpOnly || cookies[0].Path != "/" || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie: %+v", cookies)
	}
	req := httptest.NewRequest(http.MethodPost, "/upload/bytes/", bytes.NewReader(pngBytes(t)))
	req.Header.Set("Accept", "application/json")
	req.AddCookie(cookies[0])
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("cookie upload: %d", rec.Code)
	}
}
