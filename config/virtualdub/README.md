# VirtualDub configuration

VirtualDub is used only for Deshaker. Every participating file always creates
two jobs: analysis first and render second. All files are written into one
`medialang.jobs`, and VirtualDub2 is invoked once per run.

On Windows, `vdub.bat` beside the executable invokes VirtualDub with three parameters:
the configured VirtualDub executable, the absolute `medialang.jobs` path, and
`/x`. Linux continues to launch the configured Wine command directly.

`deshaker.yaml` holds the analysis (`19|1|`) and render (`19|2|`) blocks.
Codec fragments live in `codecs/`. Templates may use `Num`, `FileIndex`,
`InFile`, `OutFile` and `LogFile`. `Num` is the sequential VirtualDub job
number; `FileIndex` is the dense one-based file number after successful
preparation.

These YAML files and all `*.tpl` files are embedded in the executable at build
time. They are source resources, not files required beside a distributed
binary. Rebuild after changing them.

The runner selects codecs automatically:

- no scaling after Deshaker: `prores-pcm-mov`, directly to the final file;
- scaling after Deshaker: `huffyuv` to
  `<work_dir>/<run-id>/0001-<basename>.avi`, followed by FFmpeg.

The analysis log is unique per file at
`<work_dir>/<run-id>/0001-<basename>.log`. Paths passed to VirtualDub are
converted to Windows syntax.
