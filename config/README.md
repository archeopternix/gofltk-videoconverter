# MediaLang configuration

The application configuration has the sections `tools`, `paths` and
`processing`. Relative paths are resolved against the directory containing the
selected app YAML file.

## Tools

```yaml
tools:
  ffprobe:
    path: ffprobe
  ffmpeg:
    path: ffmpeg
  virtualdub:
    executable: wine
    arguments:
      - /opt/VirtualDub2/VirtualDub64.exe
      - /r{{.JobsFile}}
      - /x
    path_mode: wine
    wine_drive: "Z:"
    path_mappings: []
    env: {}
```

- `tools.ffprobe.path` is required.
- `tools.ffmpeg.path` is required by workflows using `ffmpeg.zscale`.
- `tools.virtualdub.executable` is required by workflows using VirtualDub.
- `arguments` is optional. Use `{{.JobsFile}}` for the jobs path. On Linux it
  is converted to a Windows-style path, its configured Wine drive prefix is
  removed, and the complete argument is wrapped in literal double quotes. On
  Windows and other operating systems the host path is substituted unchanged.
- `path_mode` is `windows` or `wine`. An empty value defaults to `windows`.
- `wine_drive` defaults to `Z:`.
- `path_mappings` can override Wine mappings, for example:

```yaml
path_mappings:
  - source: /mnt/media
    target: 'M:\media'
```

- `env` adds environment variables such as `WINEPREFIX` to VirtualDub.

## Paths

```yaml
paths:
  workflows: workflows
  avisynth_profiles: avisynth/profiles
  avisynth: /opt/avisynth/plugins64+
  virtualdub_codecs: virtualdub/codecs
  deshaker: virtualdub/deshaker.yaml
```

`workflows` is required. The remaining paths are required when their
corresponding filters or template variables are used.

## Processing

```yaml
processing:
  output_dir: ../output
  work_dir: ../work
  keep_temp_files: false
```

`output_dir` contains final user files. Scripts, per-file Deshaker logs, the
jobs file and intermediate videos are written directly below
`<work_dir>/<run-id>/`.
When `keep_temp_files` is false, that run directory is removed after success,
failure or cancellation. A successful FFmpeg step deletes its HuffYUV
intermediate immediately, including when `keep_temp_files` is true.
