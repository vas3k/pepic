package processing

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/vas3k/pepic/pepic/config"
	"github.com/vas3k/pepic/pepic/entity"
)

func TestVideoProcessing(t *testing.T) {
	old := config.App
	defer func() { config.App = old }()
	if err := config.Load("../../etc/pepic/config.yml"); err != nil {
		t.Fatal(err)
	}
	config.App.Videos.FFmpeg.TempDir = t.TempDir()
	p := filepath.Join(t.TempDir(), "clip.webm")
	if out, err := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "color=c=red:s=64x32:r=5", "-t", "0.6", p).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	data, _ := os.ReadFile(p)
	f := &entity.ProcessingFile{Filename: "example.webm", Mime: "video/webm", Bytes: data}
	v := NewVideoBackend()
	if err := v.Convert(f, "video/mp4"); err != nil {
		t.Fatal(err)
	}
	if err := v.Transcode(f, 32); err != nil {
		t.Fatal(err)
	}
	if mime, err := ValidateMedia(f.Bytes); err != nil || mime != "video/mp4" {
		t.Fatalf("%s %v", mime, err)
	}
	entries, _ := os.ReadDir(config.App.Videos.FFmpeg.TempDir)
	if len(entries) != 0 {
		t.Fatal("temporary files leaked")
	}
}
