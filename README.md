# MediaLang video converter

MediaLang reads video metadata with `ffprobe`, selects a YAML workflow, and
executes the configured AviSynth -> VirtualDub -> FFmpeg stages. AviSynth,
VirtualDub and FFmpeg are optional per workflow; at least one stage must write
the final media file.

## Configuration

MediaLang selects its default configuration by operating system:

- Windows: `config/app.windows.yaml`
- Linux: `config/app.linux.yaml`

Other operating systems are rejected. Use `-config <file>` to override the
default on Windows or Linux. Detailed settings are described in
`config/README.md` and the README files below the configuration directories.

On Windows, VirtualDub is launched through `cmd/medialang/vdub.bat`. On Linux,
the configured Wine executable and arguments are used directly.

## Usage

```text
medialang [options] <file-or-pattern> [<file-or-pattern> ...]
```

Inputs may be one file, multiple files, or quoted file patterns:

```text
go run ./cmd/medialang video.mov
go run ./cmd/medialang video1.mov video2.mp4
go run ./cmd/medialang "*.mov"
go run ./cmd/medialang "C:\Videos\*.mp4" "D:\Archive\*.m2t"
```

Use `-?`, `-h`, or `-help` to display usage. A malformed pattern or a pattern
with no matches is reported before processing begins.

## Processing

Workflow definitions are loaded from `config/workflows`. Generated AviSynth
scripts, Deshaker logs, `medialang.jobs`, and intermediate media are stored in
`<work_dir>/<run-id>/`. When `keep_temp_files` is false, this directory is
removed only after a successful run; failed, skipped, or cancelled runs retain
it for diagnostics.

VirtualDub processes all applicable files in one jobs file. Without subsequent
scaling it writes the final ProRes/PCM MOV directly. When an FFmpeg scale stage
follows, VirtualDub writes a temporary HuffYUV/PCM AVI and FFmpeg creates the
final output.

## Logging

Logs use structured `slog` output with `stage` followed by `run_id`. Supported
stages are `started`, `preparation`, `probe`, `interlace`, `deshake`, `scale`,
`file written`, and `finished`. Intermediate progress is logged at `DEBUG`, the
final summary at `INFO`, and failures at `ERROR`. Optional stages that are not
part of a selected workflow do not produce log messages.
