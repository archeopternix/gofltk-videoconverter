package assets

import "embed"

// Files contains the runtime templates and VirtualDub presets that are part of
// the application rather than user-editable deployment configuration.
//
//go:embed avisynth/profiles/*.tpl virtualdub/*.tpl virtualdub/*.yaml virtualdub/codecs/*.yaml
var Files embed.FS
