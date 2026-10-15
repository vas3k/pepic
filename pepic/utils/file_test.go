package utils

import (
	"testing"

	"github.com/vas3k/pepic/pepic/config"
)

func TestCanonicalExtensionsFollowConfig(t *testing.T) {
	old := config.App
	t.Cleanup(func() { config.App = old })
	if err := config.Load("../../etc/pepic/config.yml"); err != nil {
		t.Fatal(err)
	}
	if ext, err := ExtensionByMimeType("image/pjpeg"); err != nil || ext != ".jpg" {
		t.Fatalf("%s %v", ext, err)
	}
	config.App.CanonicalExtensions[".jpeg"] = ".photo"
	if ext, err := ExtensionByMimeType("image/jpeg"); err != nil || ext != ".photo" {
		t.Fatalf("%s %v", ext, err)
	}
	mimeType, _, allowed := config.App.Media.FormatByExtension(".photo")
	if !allowed || mimeType != "image/jpeg" {
		t.Fatal("serving does not follow canonical extensions")
	}
	delete(config.App.Media.Formats, "image/jpeg")
	if _, err := ExtensionByMimeType("image/pjpeg"); err == nil {
		t.Fatal("disabled alias still allowed")
	}
	if _, _, allowed := config.App.Media.FormatByExtension(".photo"); allowed {
		t.Fatal("disabled extension still allowed")
	}
}
