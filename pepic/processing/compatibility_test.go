package processing

import (
	"bytes"
	"github.com/vas3k/pepic/pepic/config"
	"github.com/vas3k/pepic/pepic/entity"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"testing"
)

func TestSingleFrameGIF(t *testing.T) {
	var data bytes.Buffer
	img := image.NewPaletted(image.Rect(0, 0, 32, 32), color.Palette{color.Black, color.White})
	if err := gif.Encode(&data, img, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateMedia(data.Bytes()); err != nil {
		t.Fatalf("valid static GIF rejected: %v", err)
	}
}

func TestSupportedImageRoundTrip(t *testing.T) {
	var source bytes.Buffer
	if err := png.Encode(&source, image.NewRGBA(image.Rect(0, 0, 64, 32))); err != nil {
		t.Fatal(err)
	}
	backend := NewImageBackend()
	for mimeType, format := range config.App.Media.Formats {
		if format.Processor != "vips" {
			continue
		}
		t.Run(mimeType, func(t *testing.T) {
			file := &entity.ProcessingFile{Filename: "fixture.png", Mime: "image/png", Bytes: source.Bytes()}
			if err := backend.Process(file, 32, mimeType); err != nil {
				t.Fatal(err)
			}
			actual, err := ValidateMedia(file.Bytes)
			if err != nil || actual != mimeType {
				t.Fatalf("round trip MIME = %q, error = %v", actual, err)
			}
		})
	}
}
