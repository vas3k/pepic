package config

import (
	"os"

	"github.com/caarlos0/env/v11"
	"go.yaml.in/yaml/v3"
)

func Defaults() Config {
	next := Config{Jobs: defaultJobs(), Media: defaultMedia(), Safety: defaultSafety(), CanonicalExtensions: defaultCanonicalExtensions()}
	next.Images.GIFConvert = "video/mp4"
	next.Videos.AutoConvert = next.Images.GIFConvert
	return next
}

func Load(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	next := Defaults()
	if err := env.ParseWithOptions(&next, env.Options{Environment: map[string]string{}}); err != nil {
		return err
	}
	if err := yaml.Unmarshal(data, &next); err != nil {
		return err
	}
	if err := env.ParseWithOptions(&next, env.Options{DefaultValueTagName: "unusedDefault"}); err != nil {
		return err
	}
	App = next
	return nil
}
