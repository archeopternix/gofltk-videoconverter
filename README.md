# MediaLang video converter

MediaLang selects a YAML workflow from `ffprobe` metadata and executes an
optional `AviSynth → VirtualDub → FFmpeg` pipeline. Every engine block may be
omitted. All applicable VirtualDub files are collected in one jobs file and
VirtualDub2 is invoked exactly once per run. FFmpeg executes per file and can
also consume a source or AviSynth script directly.

## Configuration

- `config/app.yaml` contains the application configuration.
- Put AviSynth scripts in `config/avisynth/profiles/*.avs.tpl`.
- Put exported VirtualDub codec blocks in `config/virtualdub/codecs/*.yaml`.
- Put the two Deshaker blocks in `config/virtualdub/deshaker.yaml`.
- Detailed configuration is documented in `config/README.md` and the README
  files in its three subdirectories.

For Wine, native media paths are converted to `Z:\...` by default. The jobs
argument uses `{{.JobsFile}}`; on Linux its converted path keeps backslashes
but omits the Wine drive prefix. Custom mounts can be declared with
`tools.virtualdub.path_mappings`, for example:

```yaml
path_mappings:
  - source: /mnt/video
    target: 'V:\video'
```

## Run

```text
go run ./cmd/medialang -config config/app.yaml video1.m2t video2.avi
```

Final archive outputs use `<basename>_processed.mov`. The supplied workflows
store ProRes HQ video with PCM audio. Files without a matching workflow, or
files that fail during preparation or FFmpeg, are skipped while later files
continue. VirtualDub always runs analysis and then Deshake, without retry.
Without subsequent scaling it writes ProRes/PCM directly; with subsequent
scaling it writes a temporary HuffYUV/PCM AVI which FFmpeg deletes after
success.
