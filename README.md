# MediaLang video converter

MediaLang selects a YAML workflow from `ffprobe` metadata, optionally creates
one AviSynth script per input and collects every successful file in one
VirtualDub2 jobs file. VirtualDub2 is invoked exactly once per run.

## Configuration

- `config/app.yaml` is the Wine/Linux example.
- `config/app.windows.yaml` is the native Windows example.
- Put AviSynth scripts in `config/avisynth/profiles/*.avs.tpl`.
- Put exported VirtualDub codec blocks in `config/virtualdub/codecs/*.yaml`.
- Put the two Deshaker blocks in `config/virtualdub/deshaker.yaml`.

For Wine, native paths are converted to `Z:\...` by default. Custom mounts can
be declared with `tools.virtualdub.path_mappings`, for example:

```yaml
path_mappings:
  - source: /mnt/video
    target: 'V:\video'
```

## Run

```text
go run ./cmd/medialang -config config/app.yaml video1.m2t video2.avi
```

Files without a matching workflow, or files that fail during preparation, are
skipped. Remaining files are added to the single VirtualDub2 batch. A failure
of that one VirtualDub2 process is not retried.
