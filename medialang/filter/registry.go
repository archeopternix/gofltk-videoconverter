package filter

import (
	"fmt"

	"github.com/archeopternix/gofltk-videoconverter/medialang/engine"
	"gopkg.in/yaml.v3"
)

type Factory func(nodeID string, config yaml.Node) (Filter, error)

type Registry struct {
	factories map[string]Factory
}

func NewRegistry() *Registry {
	return &Registry{factories: make(map[string]Factory)}
}

func DefaultRegistry() *Registry {
	registry := NewRegistry()
	registry.Register("avisynth.profile", newAviSynthProfile)
	registry.Register("virtualdub.deshaker", newDeshaker)
	registry.Register("ffmpeg.zscale", newZScale)
	return registry
}

func newZScale(nodeID string, node yaml.Node) (Filter, error) {
	var config ZScaleConfig
	if err := node.Decode(&config); err != nil {
		return nil, fmt.Errorf("decode zscale filter %q: %w", nodeID, err)
	}
	if config.Filter == "" {
		config.Filter = "spline36"
	}
	if config.PixelFormat == "" {
		config.PixelFormat = "yuv420p"
	}
	if config.Width <= 0 || config.Height <= 0 || config.VideoCodec == "" || config.AudioCodec == "" || config.Extension == "" {
		return nil, fmt.Errorf("ffmpeg.zscale %q requires width, height, video_codec, audio_codec and extension", nodeID)
	}
	if config.Extension[0] != '.' {
		config.Extension = "." + config.Extension
	}
	return &ZScale{
		base:   base{nodeID: nodeID, filterType: "ffmpeg.zscale", engineType: engine.FFmpeg},
		Config: config,
	}, nil
}

func (r *Registry) Register(filterType string, factory Factory) {
	r.factories[filterType] = factory
}

func (r *Registry) Create(nodeID, filterType string, config yaml.Node) (Filter, error) {
	factory, ok := r.factories[filterType]
	if !ok {
		return nil, fmt.Errorf("unknown filter %q", filterType)
	}
	return factory(nodeID, config)
}

func newAviSynthProfile(nodeID string, node yaml.Node) (Filter, error) {
	var config AviSynthProfileConfig
	if err := node.Decode(&config); err != nil {
		return nil, fmt.Errorf("decode avisynth profile %q: %w", nodeID, err)
	}
	if config.Profile == "" {
		return nil, fmt.Errorf("avisynth profile %q has no profile name", nodeID)
	}
	return &AviSynthProfile{
		base:   base{nodeID: nodeID, filterType: "avisynth.profile", engineType: engine.AviSynth},
		Config: config,
	}, nil
}

func newDeshaker(nodeID string, node yaml.Node) (Filter, error) {
	var config DeshakerConfig
	if err := node.Decode(&config); err != nil {
		return nil, fmt.Errorf("decode deshaker %q: %w", nodeID, err)
	}
	return &Deshaker{
		base:   base{nodeID: nodeID, filterType: "virtualdub.deshaker", engineType: engine.VirtualDub},
		Config: config,
	}, nil
}
