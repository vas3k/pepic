package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadUsesModuleDefaultsAndOverrides(t *testing.T) {
	old := App
	t.Cleanup(func() { App = old })
	filename := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(filename, []byte("global:\n  max_upload_size: 2M\ncanonical_extensions:\n  .jpeg: .photo\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Load(filename); err != nil {
		t.Fatal(err)
	}
	if err := Validate(); err != nil {
		t.Fatal(err)
	}
	if App.Safety.MaxConcurrentJobs != defaultSafety().MaxConcurrentJobs || len(App.Media.Formats) != len(defaultMedia().Formats) {
		t.Fatal("module defaults not loaded")
	}
	if App.Media.CanonicalExtension(".jpeg") != ".photo" {
		t.Fatal("canonical extension override ignored")
	}
	if err := os.WriteFile(filename, []byte("safety:\n  max_concurrent_jobs: 3\nmedia:\n  formats:\n    image/png:\n      extension: .picture\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Load(filename); err != nil {
		t.Fatal(err)
	}
	if App.Safety.MaxConcurrentJobs != 3 {
		t.Fatal("safety override ignored")
	}
	if format := App.Media.Formats["image/png"]; format.Extension != ".picture" || format.Encode == nil {
		t.Fatal("format overrides must preserve default decoder")
	}
	if err := Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestConfigurationPrecedenceAndLegacyDefaults(t *testing.T) {
	old := App
	t.Cleanup(func() { App = old })
	filename := filepath.Join(t.TempDir(), "config.yml")
	data := []byte("global:\n  max_upload_size: 123M\n  port: 19191\nimages:\n  store_originals: true\n  live_resize: false\nvideos:\n  original_length: 720\n")
	if err := os.WriteFile(filename, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := Load(filename); err != nil {
		t.Fatal(err)
	}
	if App.Global.MaxUploadSize != "123M" || App.Global.Port != 19191 || !App.Images.StoreOriginals || App.Images.LiveResize || App.Videos.OriginalLength != 720 {
		t.Fatal("YAML settings were overwritten by defaults")
	}
	if App.Media.CanonicalExtension(".jpeg") != ".jpg" || App.Images.GIFConvert == "" || App.Videos.AutoConvert == "" {
		t.Fatal("legacy configuration lost media defaults")
	}
	t.Setenv("MAX_UPLOAD_SIZE", "5M")
	t.Setenv("IMAGE_STORE_ORIGINALS", "false")
	if err := Load(filename); err != nil {
		t.Fatal(err)
	}
	if App.Global.MaxUploadSize != "5M" || App.Images.StoreOriginals {
		t.Fatal("environment did not override YAML")
	}
}

func TestRemoveAllowedFormat(t *testing.T) {
	old := App
	t.Cleanup(func() { App = old })
	filename := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(filename, []byte("media:\n  formats:\n    image/tiff: null\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Load(filename); err != nil {
		t.Fatal(err)
	}
	if err := Validate(); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := App.Media.Format("image/tiff"); ok {
		t.Fatal("removed format remains allowed")
	}
}
