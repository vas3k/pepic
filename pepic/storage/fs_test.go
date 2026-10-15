package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vas3k/pepic/pepic/config"
)

func TestStorageConfinesPathsAndSymlinks(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	secret := filepath.Join(outside, "secret")
	if err := os.WriteFile(secret, []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	backend := NewFileSystemBackend(root)
	for _, name := range []string{"../secret", "escape/secret", secret} {
		if _, err := backend.GetObject(name); err == nil {
			t.Fatalf("read escaped storage: %s", name)
		}
		if _, err := backend.PutObject(name, []byte("changed")); err == nil {
			t.Fatalf("write escaped storage: %s", name)
		}
	}
	data, err := os.ReadFile(secret)
	if err != nil || string(data) != "outside" {
		t.Fatalf("outside file changed: %s %v", data, err)
	}
}

func TestStorageAtomicReplacement(t *testing.T) {
	old := config.App
	t.Cleanup(func() { config.App = old })
	config.App.Global.MaxUploadSize = "1M"
	backend := NewFileSystemBackend(t.TempDir())
	for _, contents := range []string{"first", "replacement"} {
		if _, err := backend.PutObject("orig/media", []byte(contents)); err != nil {
			t.Fatal(err)
		}
		data, err := backend.GetObject("orig/media")
		if err != nil || string(data) != contents {
			t.Fatalf("%s %v", data, err)
		}
	}
}
