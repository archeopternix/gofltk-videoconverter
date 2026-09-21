package filter

import (
	"github.com/archeopternix/gofltk-videoconverter/medialang/engine"
	"github.com/archeopternix/gofltk-videoconverter/medialang/media"
)

type Filter interface {
	NodeID() string
	Type() string
	Engine() engine.Type
	Validate(input media.MediaSpec) (media.MediaSpec, error)
}

type base struct {
	nodeID     string
	filterType string
	engineType engine.Type
}

func (f base) NodeID() string      { return f.nodeID }
func (f base) Type() string        { return f.filterType }
func (f base) Engine() engine.Type { return f.engineType }
func (f base) Validate(input media.MediaSpec) (media.MediaSpec, error) {
	return input, nil
}

type AviSynthProfileConfig struct {
	Profile string         `yaml:"profile"`
	Preset  string         `yaml:"preset"`
	ResizeX int            `yaml:"resize_x"`
	ResizeY int            `yaml:"resize_y"`
	Values  map[string]any `yaml:"values"`
}

type AviSynthProfile struct {
	base
	Config AviSynthProfileConfig
}

type DeshakerConfig struct{}

type Deshaker struct {
	base
	Config DeshakerConfig
}

type ZScaleConfig struct {
	Width        int      `yaml:"width"`
	Height       int      `yaml:"height"`
	Filter       string   `yaml:"filter"`
	PixelFormat  string   `yaml:"pixel_format"`
	PadWidth     int      `yaml:"pad_width,omitempty"`
	PadHeight    int      `yaml:"pad_height,omitempty"`
	PadX         int      `yaml:"pad_x,omitempty"`
	PadY         int      `yaml:"pad_y,omitempty"`
	VideoCodec   string   `yaml:"video_codec"`
	CRF          *int     `yaml:"crf,omitempty"`
	Preset       string   `yaml:"preset,omitempty"`
	AudioCodec   string   `yaml:"audio_codec"`
	AudioBitrate string   `yaml:"audio_bitrate,omitempty"`
	Extension    string   `yaml:"extension"`
	ExtraArgs    []string `yaml:"extra_args,omitempty"`
}

type ZScale struct {
	base
	Config ZScaleConfig
}
