package handler

import (
	"bytes"
	"image"
	"image/jpeg"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/vas3k/pepic/pepic/config"
	"github.com/vas3k/pepic/pepic/storage"
	"github.com/vas3k/pepic/pepic/utils"
)

func TestLegacyMediaDownloads(t *testing.T) {
	for _, split := range []int{2, 3} {
		t.Run(strconv.Itoa(split), func(t *testing.T) {
			e, dir := setupUpload(t)
			config.App.Global.FileTreeSplitChars = split
			config.App.Images.LiveResize = false
			h := &PepicHandler{Storage: storage.NewStorage(storage.NewFileSystemBackend(dir))}
			e.GET("/:length/:name", h.GetResizedFile)
			var encoded bytes.Buffer
			if err := jpeg.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 2, 2)), nil); err != nil {
				t.Fatal(err)
			}
			data := encoded.Bytes()
			backend := storage.NewFileSystemBackend(dir)
			for _, name := range []string{"4c.jpg", "i4.jpg", "2vv.jpg", "Old_image-1.jpg"} {
				if _, err := backend.PutObject("orig/"+utils.CanonizeFileName(name), data); err != nil {
					t.Fatal(err)
				}
				for _, prefix := range []string{"/", "/full/", "/200/"} {
					rec := httptest.NewRecorder()
					e.ServeHTTP(rec, httptest.NewRequest("GET", prefix+name, nil))
					if rec.Code != 200 || !bytes.Equal(rec.Body.Bytes(), data) {
						t.Fatalf("%s%s: HTTP %d", prefix, name, rec.Code)
					}
				}
			}
		})
	}
}
