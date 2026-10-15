package processing

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/vas3k/pepic/pepic/config"
	"github.com/vas3k/pepic/pepic/entity"
)

func TestImageConversionResizesAndFlattens(t *testing.T) {
	var input bytes.Buffer
	original := image.NewNRGBA(image.Rect(0, 0, 64, 32))
	original.Set(32, 16, color.NRGBA{R: 255, A: 255})
	if err := png.Encode(&input, original); err != nil {
		t.Fatal(err)
	}
	file := &entity.ProcessingFile{Filename: "test.png", Mime: "image/png", Bytes: input.Bytes()}
	if err := NewImageBackend().Process(file, 32, "image/jpeg"); err != nil {
		t.Fatal(err)
	}
	if file.Filename != "test.jpg" || file.Mime != "image/jpeg" {
		t.Fatalf("%s %s", file.Filename, file.Mime)
	}
	result, err := jpeg.Decode(bytes.NewReader(file.Bytes))
	if err != nil {
		t.Fatal(err)
	}
	if result.Bounds().Dx() != 32 || result.Bounds().Dy() != 16 {
		t.Fatal(result.Bounds())
	}
	red, green, blue, _ := result.At(0, 0).RGBA()
	if red < 60000 || green < 60000 || blue < 60000 {
		t.Fatal("transparent pixel was not flattened to white")
	}
}

func TestImageConversionHonorsFormatPolicy(t *testing.T) {
	old := config.App.Media.Formats
	config.App.Media.Formats = make(map[string]config.MediaFormat)
	for key, value := range old {
		config.App.Media.Formats[key] = value
	}
	t.Cleanup(func() { config.App.Media.Formats = old })
	delete(config.App.Media.Formats, "image/jpeg")
	file := &entity.ProcessingFile{Filename: "test.png", Mime: "image/png"}
	if err := NewImageBackend().Process(file, 32, "image/jpeg"); err == nil {
		t.Fatal("disabled conversion accepted")
	}
}
