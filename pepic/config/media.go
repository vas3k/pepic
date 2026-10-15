package config

import (
	"encoding/hex"
	"fmt"
	"mime"
	"path"
	"sort"
	"strings"

	"github.com/davidbyttow/govips/v2/vips"
	"go.yaml.in/yaml/v3"
)

type MediaFormat struct {
	Kind            string                                                                   `yaml:"kind"`
	Processor       string                                                                   `yaml:"processor"`
	Extension       string                                                                   `yaml:"extension"`
	Aliases         []string                                                                 `yaml:"aliases"`
	ImageType       vips.ImageType                                                           `yaml:"-"`
	Encode          func(*vips.ImageRef, ImageEncoding) ([]byte, *vips.ImageMetadata, error) `yaml:"-"`
	AllowStillImage bool                                                                     `yaml:"allow_still_image"`
	Animated        bool                                                                     `yaml:"animated"`
	GoDecode        bool                                                                     `yaml:"go_decode"`
	FlattenAlpha    bool                                                                     `yaml:"flatten_alpha"`
	Demuxer         string                                                                   `yaml:"demuxer"`
	InputArgs       []string                                                                 `yaml:"input_args"`
	UseMovFlags     bool                                                                     `yaml:"use_mov_flags"`
	Signatures      []Signature                                                              `yaml:"signatures"`
}

type Signature struct {
	Offset int    `yaml:"offset"`
	Hex    string `yaml:"hex"`
}

type MediaConfig struct {
	Formats map[string]MediaFormat `yaml:"formats"`
}

func (m MediaConfig) Format(mimeType string) (string, MediaFormat, bool) {
	if f, ok := m.Formats[mimeType]; ok {
		return mimeType, f, true
	}
	for canonical, f := range m.Formats {
		for _, alias := range f.Aliases {
			if mimeType == alias {
				return canonical, f, true
			}
		}
	}
	return "", MediaFormat{}, false
}

func (m MediaConfig) CanonicalExtension(extension string) string {
	if canonical, ok := App.CanonicalExtensions[extension]; ok {
		return canonical
	}
	return extension
}

func (m MediaConfig) FormatByExtension(extension string) (string, MediaFormat, bool) {
	extension = m.CanonicalExtension(strings.ToLower(extension))
	for mimeType, f := range m.Formats {
		if m.CanonicalExtension(f.Extension) == extension {
			return mimeType, f, true
		}
	}
	return "", MediaFormat{}, false
}

func (m MediaConfig) MIMETypes() []string {
	types := make([]string, 0, len(m.Formats))
	for mimeType := range m.Formats {
		types = append(types, mimeType)
	}
	sort.Strings(types)
	return types
}

func (m MediaConfig) Validate() error {
	if len(m.Formats) == 0 {
		return fmt.Errorf("media.formats must contain at least one format")
	}
	seenMIME, seenExtensions := map[string]bool{}, map[string]bool{}
	seenImageTypes := map[vips.ImageType]bool{}
	for mimeType, f := range m.Formats {
		for _, name := range append([]string{mimeType}, f.Aliases...) {
			parsed, params, err := mime.ParseMediaType(name)
			if err != nil || parsed != name || len(params) != 0 || seenMIME[name] {
				return fmt.Errorf("invalid or duplicate media MIME %q", name)
			}
			seenMIME[name] = true
		}
		if f.Kind != "image" && f.Kind != "video" {
			return fmt.Errorf("invalid kind for %s", mimeType)
		}
		if !validExtension(f.Extension) {
			return fmt.Errorf("invalid extension for %s", mimeType)
		}
		ext := m.CanonicalExtension(f.Extension)
		if seenExtensions[ext] {
			return fmt.Errorf("duplicate canonical extension %q", ext)
		}
		seenExtensions[ext] = true
		switch f.Processor {
		case "vips":
			if f.Kind != "image" || f.ImageType == vips.ImageTypeUnknown || f.Animated || f.Encode == nil {
				return fmt.Errorf("invalid vips format %s", mimeType)
			}
		case "ffmpeg":
			if f.Demuxer == "" {
				return fmt.Errorf("missing demuxer for %s", mimeType)
			}
		default:
			return fmt.Errorf("invalid processor for %s", mimeType)
		}
		if f.ImageType != vips.ImageTypeUnknown {
			if seenImageTypes[f.ImageType] {
				return fmt.Errorf("duplicate image type %v", f.ImageType)
			}
			seenImageTypes[f.ImageType] = true
		}
		for _, signature := range f.Signatures {
			data, err := hex.DecodeString(signature.Hex)
			if err != nil || len(data) == 0 || signature.Offset < 0 {
				return fmt.Errorf("invalid signature for %s", mimeType)
			}
		}
	}
	for alias, canonical := range App.CanonicalExtensions {
		if !validExtension(alias) || !validExtension(canonical) || alias == canonical {
			return fmt.Errorf("invalid canonical extension %q", alias)
		}
		if _, chained := App.CanonicalExtensions[canonical]; chained {
			return fmt.Errorf("canonical extension chains are not allowed: %q", alias)
		}
	}
	return nil
}

func validExtension(extension string) bool {
	return len(extension) > 1 && extension == strings.ToLower(extension) && path.Ext("file"+extension) == extension && !strings.ContainsAny(extension, "/\\\x00")
}

type ImageEncoding struct {
	Quality     int
	Compression int
}

func (m *MediaConfig) UnmarshalYAML(node *yaml.Node) error {
	var overrides struct {
		Formats map[string]yaml.Node `yaml:"formats"`
	}
	if err := node.Decode(&overrides); err != nil {
		return err
	}
	if m.Formats == nil {
		m.Formats = make(map[string]MediaFormat)
	}
	for mimeType, value := range overrides.Formats {
		if value.Tag == "!!null" {
			delete(m.Formats, mimeType)
			continue
		}
		format := m.Formats[mimeType]
		if err := value.Decode(&format); err != nil {
			return err
		}
		m.Formats[mimeType] = format
	}
	return nil
}
