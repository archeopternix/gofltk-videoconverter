# AviSynth profiles

Profiles are named Go `text/template` definitions below `profiles/` with the
suffix `.avs.tpl`. Each file must define `{{define "Avisynth"}}`; unknown
template variables are errors.

Built-in values are `AvisynthPath`, `InFile`, `Deinterlace`, `ConvertYV`,
`Preset`, `ResizeX`, `ResizeY`, `Width`, `Height`, `FPS`, `FieldOrder`,
`PixelFormat`, `ColorSpace` and `ColorFamily`. Additional typed values may be
provided through `config.values`.

`AvisynthPath` and `InFile` are already converted according to VirtualDub's
`path_mode` (`windows` by default, `wine` explicitly on Linux). A workflow can
configure a profile as follows:

```yaml
config:
  profile: hdv-25i
  preset: Slower
  resize_x: 1920
  resize_y: 1080
```

The generated script is stored as
`<work_dir>/<run-id>/0001-<basename>.avs` and becomes the actual input for the
following VirtualDub or FFmpeg stage. Convert to YV12 before filters which
require YV12, then deinterlace/crop and resize in the profile.
