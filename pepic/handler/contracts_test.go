package handler

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vas3k/pepic/pepic/config"
	"github.com/vas3k/pepic/pepic/jobs"
)

func TestThirdPartyMultipartContracts(t *testing.T) {
	for _, client := range []string{"EasyMDE", "Django"} {
		t.Run(client, func(t *testing.T) {
			e, _ := setupUpload(t)
			config.App.Global.SecretCode = "integration-secret"
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			if client == "EasyMDE" {
				if err := writer.WriteField("code", "integration-secret"); err != nil {
					t.Fatal(err)
				}
			}
			file, err := writer.CreateFormFile("media", "upload.png")
			if err != nil {
				t.Fatal(err)
			}
			file.Write(pngBytes(t))
			writer.Close()
			url := "/upload/multipart/"
			if client == "Django" {
				url += "?code=integration-secret"
			}
			result := request(e, url, &body, writer.FormDataContentType(), false)
			if result.Code != http.StatusCreated {
				t.Fatalf("%s HTTP %d: %s", client, result.Code, result.Body.String())
			}
			var payload map[string]json.RawMessage
			if err := json.Unmarshal(result.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			var urls []string
			if err := json.Unmarshal(payload["uploaded"], &urls); err != nil || len(urls) != 1 || urls[0] == "" {
				t.Fatalf("%s response contract broken: %s", client, result.Body.String())
			}
			if len(payload) != 1 {
				t.Fatalf("unexpected async response fields: %s", result.Body.String())
			}
		})
	}
}

func TestGlobalJobListAuthenticationAndVisibility(t *testing.T) {
	e, h, _ := setupJobs(t)
	first := jobRequest(t, e, "/upload/jobs/bytes/", bytes.NewReader(pngBytes(t)), "image/png")
	second := jobRequest(t, e, "/upload/jobs/bytes/", bytes.NewReader(pngBytes(t)), "image/png")
	config.App.Global.SecretCode = "shared-secret"
	get := func(url string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Accept", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec
	}
	if rec := get("/jobs/"); rec.Code != 401 {
		t.Fatalf("public queue listing: %d", rec.Code)
	}
	rec := get("/jobs/?code=shared-secret")
	if rec.Code != 200 || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("listing: %d %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Jobs        []jobs.Job `json:"jobs"`
		PollSeconds int        `json:"poll_after_seconds"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Jobs) != 2 || payload.Jobs[0].ID != second.ID || payload.Jobs[1].ID != first.ID || payload.PollSeconds != config.App.Jobs.PollSeconds {
		t.Fatalf("jobs missing from global list: %+v", payload)
	}
	h.Jobs.Start()
}
