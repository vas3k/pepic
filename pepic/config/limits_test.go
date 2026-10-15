package config

import "testing"

func TestUploadLimit(t *testing.T) {
	old := App
	t.Cleanup(func() { App = old })
	for _, value := range []string{"", "0", "-1M", "nonsense"} {
		App.Global.MaxUploadSize = value
		if _, err := UploadLimit(); err == nil {
			t.Fatalf("accepted %q", value)
		}
	}
	App.Global.MaxUploadSize = "750M"
	if n, err := UploadLimit(); err != nil || n != 750000000 {
		t.Fatalf("%d %v", n, err)
	}
}

func TestSafetyValidation(t *testing.T) {
	old := App
	t.Cleanup(func() { App = old })
	if err := Validate(); err != nil {
		t.Fatal(err)
	}
	App.Safety.MaxConcurrentJobs = 0
	if err := Validate(); err == nil {
		t.Fatal("zero worker limit accepted")
	}
}

func TestMediaPolicyValidation(t *testing.T) {
	if err := App.Media.Validate(); err != nil {
		t.Fatal(err)
	}
	policy := MediaConfig{Formats: map[string]MediaFormat{}}
	for key, value := range App.Media.Formats {
		policy.Formats[key] = value
	}
	policy.Formats["image/x-duplicate"] = policy.Formats["image/png"]
	if err := policy.Validate(); err == nil {
		t.Fatal("ambiguous decoder policy accepted")
	}
}
