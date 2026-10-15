package template

import (
	"os"
	"strings"
	"testing"
)

func TestAssetURLChangesWithContents(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile("upload.js", []byte("first"), 0600); err != nil {
		t.Fatal(err)
	}
	first := assetURL("/upload.js")
	if !strings.HasPrefix(first, "/upload.js?v=") || assetURL("/upload.js") != first {
		t.Fatalf("unstable asset URL: %s", first)
	}
	if err := os.WriteFile("upload.js", []byte("second"), 0600); err != nil {
		t.Fatal(err)
	}
	if assetURL("/upload.js") == first {
		t.Fatal("changed asset reused cached URL")
	}
	if assetURL("/missing.js") != "/missing.js" {
		t.Fatal("missing asset URL changed")
	}
}
