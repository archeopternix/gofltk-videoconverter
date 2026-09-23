# MediaLang video converter

MediaLang processes video files and enhances them to be used in a NLE (best DaVinci Resolve).

Based on the source file parameters the best matching workflow will be selected and depending on the configured settings the video will be:

- De-Interlaced
- De-shaked
- Scaled
- Saved in ProRes format

It reads video metadata with `ffprobe`, selects a YAML workflow, and
executes the configured AviSynth -> VirtualDub -> FFmpeg stages. AviSynth,
VirtualDub and FFmpeg are optional per workflow; at least one stage must write
the final media file.

## Configuration

MediaLang selects its default configuration by operating system:

- Windows: `config/app.windows.yaml`
- Linux: `config/app.linux.yaml`

Other operating systems are rejected. Use `-config <file>` to override the
default on Windows or Linux. The default and any relative `-config` path are
resolved from the directory containing the executable, so the program does not
depend on its current working directory. Detailed settings are described in
`config/README.md` and the README files below the configuration directories.

On Windows, VirtualDub is launched through `vdub.bat` beside the executable.
On Linux, the configured Wine executable and arguments are used directly.

## Usage

```text
medialang [options] <file-or-pattern> [<file-or-pattern> ...]
```

Inputs may be one file, multiple files, or quoted file patterns:

```text
videoconverter.exe video.mov
videoconverter.exe video1.mov video2.mp4
videoconverter.exe "*.mov"
videoconverter.exe "C:\Videos\*.mp4" "D:\Archive\*.m2t"
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

Missing, unknown, or unrecognized scan/field-order metadata is treated as
progressive. Deinterlacing requires a known interlaced scan and field order.

A file error skips that file's remaining workflow stages; other files continue.
VirtualDub analysis and render remain in the shared external batch. After it
finishes, every participating output is checked before any FFmpeg stage starts.
Outputs must be nonempty regular files, newly created or updated, with video
metadata readable by ffprobe. Invalid outputs are skipped individually, even
when VirtualDub reports a batch error; verified outputs can still proceed.
Per-file errors are collected and reported after processing, and the command
exits with status 1 if the run had errors.

## Logging

Logs use structured `slog` output with `stage` followed by `run_id`. Supported
stages are `started`, `preparation`, `probe`, `interlace`, `deshake`, `scale`,
`file written`, and `finished`. Intermediate progress is logged at `DEBUG`, the
final summary at `INFO`, and failures at `ERROR`. Optional stages that are not
part of a selected workflow do not produce log messages.

## Windows distribution

Run the PowerShell build script from any directory:

```powershell
./scripts/build-windows.ps1
```

This creates `dist/videoconverter.zip`. The archive contains the Windows AMD64
binary, `vdub.bat`, `config/app.windows.yaml`, and the external workflow YAML
files. It does not include external tools; edit `config/app.windows.yaml` after
unpacking to point to FFmpeg, VirtualDub2, and AviSynth on the target PC.

To create a bundle with tools, supply a directory with this layout:

```text
portable-tools/
  ffmpeg/bin/ffmpeg.exe
  ffmpeg/bin/ffprobe.exe
  VirtualDub2/vdub64.exe
  AviSynth+/plugins64+/
```

```powershell
./scripts/build-windows.ps1 -ToolsDirectory C:\portable-tools
```

The complete tools directory is copied to `tools/` in the archive and the
packaged app configuration uses relative paths to it. The source
`config/app.windows.yaml` is never changed. AviSynth and VirtualDub templates,
Deshaker settings, and VirtualDub codec presets are embedded in the executable;
changing those source files requires rebuilding the binary.
