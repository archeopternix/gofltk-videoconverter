# Workflows

Matching uses the optional fields `interlaced`, `width`, `height`, `fps`,
`fps_tolerance`, `pixel_format`, `color_space` and `codec`. `fps` accepts one
number or a list, for example `[25, 29.97, 30, 50, 59.94, 60]`.

The available filters are `avisynth.profile`, `virtualdub.deshaker` and
`ffmpeg.zscale`. Each is optional and may occur at most once, but at least one
must be selected. Their order is always AviSynth → VirtualDub → FFmpeg.
AviSynth alone is not a final writer; FFmpeg can read either a source or a
generated `.avs` directly.

When a workflow specifies `interlaced: false`, media with an unknown scan type
is treated as non-interlaced. An unknown scan type does not match
`interlaced: true`.

VirtualDub encoding is selected implicitly. A Deshaker workflow without a
later zscale step writes ProRes/PCM MOV. If zscale follows, VirtualDub writes a
temporary HuffYUV/PCM AVI and FFmpeg creates ProRes HQ (`prores_ks`, profile 3,
`yuv422p10le`) with `pcm_s24le` audio.

`ffmpeg.zscale.extra_args` adds output arguments immediately before the output
file. It can enforce a constant output frame rate, for example
`["-r", "30", "-fps_mode", "cfr"]`.

```yaml
workflow:
  - id: prepare
    filter: avisynth.profile
    config:
      profile: dv-25i
      preset: Slower
      resize_x: 720
      resize_y: 576
  - id: stabilize
    filter: virtualdub.deshaker
    config: {}
  - id: final-scale
    filter: ffmpeg.zscale
    config:
      width: 1440
      height: 1080
      filter: spline36
      pixel_format: yuv422p10le
      pad_width: 1920
      pad_height: 1080
      pad_x: 240
      pad_y: 0
      video_codec: prores_ks
      audio_codec: pcm_s24le
      extension: .mov
      extra_args: ["-profile:v", "3"]
```
