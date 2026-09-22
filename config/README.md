# MediaLang configuration

The application configuration has the sections `tools`, `paths` and
`processing`. Relative paths are resolved against the directory containing the
selected app YAML file.

The default app YAML is located relative to the executable. A relative
`-config` argument is also interpreted relative to the executable, not the
current working directory. Tool values containing a slash are treated as paths
and resolved against the app YAML directory; bare commands such as `ffprobe`
or `wine` continue to use `PATH`.

By default, MediaLang loads `config/app.windows.yaml` on Windows and
`config/app.linux.yaml` on Linux. Other operating systems are rejected. The
`-config` flag can select a different configuration file on a supported system.

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
      - /s{{.JobsFile}}
      - /x
    path_mode: wine
    wine_drive: "Z:"
    path_mappings: []
    env: {}
```

- `tools.ffprobe.path` is required.
- `tools.ffmpeg.path` is required by workflows using `ffmpeg.zscale`.
- `tools.virtualdub.executable` is required by workflows using VirtualDub.
- On Windows, `arguments` is ignored because `vdub.bat` beside the executable receives
  the VirtualDub executable, absolute jobs path, and `/x` directly.
- On Linux, `arguments` is passed to the configured Wine executable. Use
  `{{.JobsFile}}` where the absolute host path to `medialang.jobs` is required.
- `path_mode` is `windows` or `wine`. An empty value defaults to `windows`.
- `wine_drive` defaults to `Z:`.
- `path_mappings` controls paths written inside AviSynth and VirtualDub jobs
  files and can override Wine mappings, for example:

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
  avisynth: /opt/avisynth/plugins64+
```

`workflows` is required and remains external so workflows can be edited without
rebuilding. `avisynth` is the plugin directory written into generated scripts.
AviSynth profiles, VirtualDub jobs templates, Deshaker settings, and codec
presets are embedded in the executable and therefore have no app YAML paths.

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
When `keep_temp_files` is false, that run directory is removed only after a
successful run. Failed, skipped or cancelled runs retain their files for
diagnostics. A successful FFmpeg step deletes its HuffYUV intermediate
immediately, including when `keep_temp_files` is true.
