package virtualdub

import (
	"fmt"
	"path/filepath"

	embedded "github.com/archeopternix/gofltk-videoconverter/config"
	"gopkg.in/yaml.v3"
)

type CodecPreset struct {
	ID              string `yaml:"id"`
	Extension       string `yaml:"extension"`
	ContainerScript string `yaml:"container_script"`
	VideoScript     string `yaml:"video_script"`
	AudioScript     string `yaml:"audio_script"`
	SaveScript      string `yaml:"save_script"`
}

type DeshakerTemplate struct {
	PluginName     string `yaml:"plugin_name"`
	AnalysisScript string `yaml:"analysis_script"`
	RenderScript   string `yaml:"render_script"`
}

type ConfigStore struct{}

func (s ConfigStore) LoadCodec(id string) (*CodecPreset, error) {
	filename := "virtualdub/codecs/" + filepath.Base(id) + ".yaml"
	data, err := embedded.Files.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read VirtualDub codec %q: %w", id, err)
	}
	var preset CodecPreset
	if err := yaml.Unmarshal(data, &preset); err != nil {
		return nil, fmt.Errorf("parse VirtualDub codec %q: %w", id, err)
	}
	if preset.ID == "" {
		preset.ID = id
	}
	if preset.Extension == "" {
		return nil, fmt.Errorf("VirtualDub codec %q has no extension", id)
	}
	if preset.Extension[0] != '.' {
		preset.Extension = "." + preset.Extension
	}
	return &preset, nil
}

func (s ConfigStore) LoadDeshaker() (*DeshakerTemplate, error) {
	data, err := embedded.Files.ReadFile("virtualdub/deshaker.yaml")
	if err != nil {
		return nil, fmt.Errorf("read Deshaker config: %w", err)
	}
	var config DeshakerTemplate
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse Deshaker config: %w", err)
	}
	return &config, nil
}
