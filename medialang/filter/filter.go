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
	Profile string            `yaml:"profile"`
	Values  map[string]string `yaml:"values"`
}

type AviSynthProfile struct {
	base
	Config AviSynthProfileConfig
}

type DeshakerConfig struct {
	ReuseAnalysis bool `yaml:"reuse_analysis"`
	ForceAnalysis bool `yaml:"force_analysis"`
}

type Deshaker struct {
	base
	Config DeshakerConfig
}

type EncodeConfig struct {
	Preset string `yaml:"preset"`
}

type Encode struct {
	base
	Config EncodeConfig
}
