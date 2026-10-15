package processing

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/vas3k/pepic/pepic/config"
)

func TestDimensionLimits(t *testing.T) {
	for _, v := range [][2]int{{0, 2}, {-1, 5}, {100000, 100000}, {10000, 10000}} {
		if validDimensions(v[0], v[1], config.App.Safety.MaxImagePixels) {
			t.Fatalf("accepted %v", v)
		}
	}
	if !validDimensions(4000, 3000, config.App.Safety.MaxImagePixels) {
		t.Fatal("normal photo rejected")
	}
}

func TestRealVideoValidation(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg unavailable")
	}
	for _, ext := range []string{"mp4", "mov", "webm", "gif", "avi", "ogv"} {
		t.Run(ext, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "clip."+ext)
			args := []string{"-v", "error", "-f", "lavfi", "-i", "color=c=red:s=32x32:r=5", "-t", "0.6"}
			if ext == "mp4" || ext == "mov" {
				args = append(args, "-c:v", "libx264", "-pix_fmt", "yuv420p")
			}
			args = append(args, p)
			if out, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
				t.Fatalf("fixture: %v %s", err, out)
			}
			data, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			mime, err := ValidateMedia(data)
			if err != nil {
				t.Fatalf("valid %s: %v", ext, err)
			}
			if mime == "" {
				t.Fatal("empty MIME")
			}
			if _, err := ValidateMedia(data[:32]); err == nil {
				t.Fatal("truncated video accepted")
			}
		})
	}
}

func TestAudioOnlyMP4Rejected(t *testing.T) {
	p := filepath.Join(t.TempDir(), "audio.mp4")
	if out, err := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "sine=frequency=1000", "-t", "0.2", "-c:a", "aac", p).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	data, _ := os.ReadFile(p)
	if _, err := ValidateMedia(data); err == nil {
		t.Fatal("audio-only MP4 accepted")
	}
}

func TestPixelBombHeaderRejected(t *testing.T) {
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	data := b.Bytes()
	binary.BigEndian.PutUint32(data[16:20], 100000)
	binary.BigEndian.PutUint32(data[20:24], 100000)
	binary.BigEndian.PutUint32(data[29:33], crc32.ChecksumIEEE(data[12:29]))
	if _, err := ValidateMedia(data); err == nil {
		t.Fatal("oversized image dimensions accepted")
	}
}

func TestConfiguredProbeOutputLimit(t *testing.T) {
	old := config.App.Safety.ProbeOutputBytes
	t.Cleanup(func() { config.App.Safety.ProbeOutputBytes = old })
	config.App.Safety.ProbeOutputBytes = 4
	var output cappedOutput
	if _, err := output.Write([]byte("1234")); err != nil {
		t.Fatal(err)
	}
	if _, err := output.Write([]byte("5")); err == nil {
		t.Fatal("probe output limit ignored")
	}
}
