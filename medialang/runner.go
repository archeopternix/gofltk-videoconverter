package medialang

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/archeopternix/gofltk-videoconverter/medialang/config"
	"github.com/archeopternix/gofltk-videoconverter/medialang/engine"
	"github.com/archeopternix/gofltk-videoconverter/medialang/engine/avisynth"
	ffmpegengine "github.com/archeopternix/gofltk-videoconverter/medialang/engine/ffmpeg"
	"github.com/archeopternix/gofltk-videoconverter/medialang/engine/virtualdub"
	"github.com/archeopternix/gofltk-videoconverter/medialang/filter"
	"github.com/archeopternix/gofltk-videoconverter/medialang/media"
	"github.com/archeopternix/gofltk-videoconverter/medialang/probe"
	"github.com/archeopternix/gofltk-videoconverter/medialang/workflow"
)

type FileStatus string

const (
	StatusCompleted FileStatus = "completed"
	StatusSkipped   FileStatus = "skipped"
	StatusFailed    FileStatus = "failed"
	StatusCancelled FileStatus = "cancelled"
)

const (
	stageStarted     = "started"
	stagePreparation = "preparation"
	stageProbe       = "probe"
	stageInterlace   = "interlace"
	stageDeshake     = "deshake"
	stageScale       = "scale"
	stageFileWritten = "file written"
	stageFinished    = "finished"
)

type FileResult struct {
	Input  string
	Output string
	Stage  string
	Status FileStatus
	Error  error
}

type BatchResult struct {
	JobsFile string
	Files    []FileResult
}

type Runner struct {
	Config   *config.App
	Files    []string
	Probe    probe.Probe
	Registry *filter.Registry
	Logger   *slog.Logger
	RunID    string
}

type plannedFile struct {
	fileIndex        int
	input            media.Artifact
	profile          *filter.AviSynthProfile
	deshaker         *filter.Deshaker
	deshakerTemplate *virtualdub.DeshakerTemplate
	codec            *virtualdub.CodecPreset
	ffmpegScale      *filter.ZScale
	virtualDubOutput string
	finalOutput      string
	current          media.Artifact
	readyForFFmpeg   bool
	result           *FileResult
}

func NewRunner(app *config.App, files []string, runID string) *Runner {
	return &Runner{
		Config:   app,
		Files:    append([]string(nil), files...),
		Probe:    probe.FFProbe{Executable: app.Tools.FFprobe.Path},
		Registry: filter.DefaultRegistry(),
		Logger:   slog.Default(),
		RunID:    runID,
	}
}

func (r *Runner) Run(ctx context.Context) (result *BatchResult, runErr error) {
	runID := r.RunID
	if runID == "" {
		runID = time.Now().Format("20060102T150405.000")
	}
	baseLogger := r.Logger
	if baseLogger == nil {
		baseLogger = slog.Default()
	}
	result = &BatchResult{Files: make([]FileResult, len(r.Files))}
	for i, filename := range r.Files {
		result.Files[i] = FileResult{Input: filename, Status: StatusSkipped}
	}
	startedAt := time.Now()
	workDir := filepath.Join(r.Config.Processing.WorkDir, runID)
	workDirCreated := false
	workDirPreserved := false
	stageLogger(baseLogger, stageStarted, runID).Debug("run started", "files", len(r.Files), "work_directory", workDir)
	defer func() {
		if workDirCreated {
			workDirPreserved = r.Config.Processing.KeepFiles || runErr != nil || batchHasErrors(result)
			if !workDirPreserved {
				if err := os.RemoveAll(workDir); err != nil {
					workDirPreserved = true
					stageLogger(baseLogger, stageFinished, runID).Warn("work directory cleanup failed", "directory", workDir, "error", err)
				}
			}
		}
		summary := summarize(result)
		stageLogger(baseLogger, stageFinished, runID).Info("run finished",
			"result", finalResult(summary, runErr),
			"duration", time.Since(startedAt),
			"completed", summary.completed,
			"skipped", summary.skipped,
			"failed", summary.failed,
			"cancelled", summary.cancelled,
			"work_directory", workDir,
			"work_directory_preserved", workDirPreserved,
		)
	}()

	catalog, err := workflow.LoadCatalog(r.Config.Paths.Workflows)
	if err != nil {
		return result, err
	}
	store := virtualdub.ConfigStore{CodecsDir: r.Config.Paths.VirtualDubCodecs, DeshakerFile: r.Config.Paths.Deshaker}
	pather := virtualdub.NewPathConverter(r.Config.Tools.VirtualDub)
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return result, fmt.Errorf("create run directory %q: %w", workDir, err)
	}
	workDirCreated = true
	if err := os.MkdirAll(r.Config.Processing.OutputDir, 0o755); err != nil {
		return result, fmt.Errorf("create output directory %q: %w", r.Config.Processing.OutputDir, err)
	}

	planned := make([]*plannedFile, 0, len(r.Files))
	outputs := make(map[string]struct{})
	for i, filename := range r.Files {
		if err := ctx.Err(); err != nil {
			markRemainingCancelled(result, i, err)
			return result, err
		}
		file, err := r.planFile(ctx, filename, &result.Files[i], catalog, store, baseLogger, runID)
		if err != nil {
			skipFile(baseLogger, runID, &result.Files[i], err)
			continue
		}
		key := strings.ToLower(filepath.Clean(file.finalOutput))
		if _, exists := outputs[key]; exists {
			skipFile(baseLogger, runID, &result.Files[i], atStage(stagePreparation, fmt.Errorf("output path is already used in this run: %s", file.finalOutput)))
			continue
		}
		outputs[key] = struct{}{}
		planned = append(planned, file)
	}

	avsCompiler := avisynth.Compiler{ProfilesDir: r.Config.Paths.AviSynthProfiles, AviSynthPath: r.Config.Paths.AviSynth, Pather: pather}
	prepared := make([]*plannedFile, 0, len(planned))
	for _, file := range planned {
		if err := ctx.Err(); err != nil {
			file.result.Status, file.result.Stage, file.result.Error = StatusCancelled, stageInterlace, err
			continue
		}
		file.fileIndex = len(prepared) + 1
		file.current = file.input
		if file.profile != nil {
			avsPath := filepath.Join(workDir, indexedName(file.fileIndex, file.input.Path, ".avs"))
			started := time.Now()
			artifact, err := avsCompiler.Compile(file.current, file.profile, avsPath)
			if err != nil {
				skipFile(baseLogger, runID, file.result, atStage(stageInterlace, err))
				continue
			}
			file.current = artifact
			stageLogger(baseLogger, stageInterlace, runID).Debug("AviSynth file written",
				"file", file.input.Path,
				"path", avsPath,
				"duration", time.Since(started),
			)
		}
		prepared = append(prepared, file)
	}

	jobs := make([]virtualdub.Job, 0, len(prepared))
	virtualDubFiles := make([]*plannedFile, 0, len(prepared))
	for _, file := range prepared {
		if file.deshaker == nil {
			file.readyForFFmpeg = file.ffmpegScale != nil
			continue
		}
		logPath := filepath.Join(workDir, indexedName(file.fileIndex, file.input.Path, ".log"))
		if file.ffmpegScale != nil {
			file.virtualDubOutput = filepath.Join(workDir, indexedName(file.fileIndex, file.input.Path, ".avi"))
		} else {
			file.virtualDubOutput = file.finalOutput
		}
		jobs = append(jobs, virtualdub.Job{FileIndex: file.fileIndex, InputPath: file.current.Path, OutputPath: file.virtualDubOutput, LogPath: logPath, Deshaker: file.deshakerTemplate, Codec: file.codec})
		virtualDubFiles = append(virtualDubFiles, file)
	}

	if len(jobs) > 0 {
		jobsFile := filepath.Join(workDir, "medialang.jobs")
		result.JobsFile = jobsFile
		builder := virtualdub.JobsBuilder{Pather: pather, TemplateDir: filepath.Dir(r.Config.Paths.Deshaker)}
		jobCount, err := builder.Write(jobsFile, jobs)
		if err != nil {
			stageErr := atStage(stagePreparation, err)
			stageLogger(baseLogger, stagePreparation, runID).Error("VirtualDub jobs file failed", "path", jobsFile, "error", err)
			for _, file := range virtualDubFiles {
				setFailure(file.result, stageErr)
			}
		} else {
			stageLogger(baseLogger, stagePreparation, runID).Debug("VirtualDub jobs file written", "path", jobsFile, "jobs", jobCount, "files", len(virtualDubFiles))
			deshakeLogger := stageLogger(baseLogger, stageDeshake, runID)
			vdubRunner := virtualdub.Runner{Tool: r.Config.Tools.VirtualDub, Logger: deshakeLogger}
			deshakeLogger.Debug("VirtualDub started",
				"executable", r.Config.Tools.VirtualDub.Executable,
				"jobs_file", jobsFile,
				"files", len(virtualDubFiles),
			)
			started := time.Now()
			virtualDubResult, vdubErr := vdubRunner.Run(ctx, jobsFile)
			duration := time.Since(started)
			if vdubErr != nil {
				stageErr := atStage(stageDeshake, vdubErr)
				deshakeLogger.Error("VirtualDub failed",
					"duration", duration,
					"exit_code", virtualDubResult.ExitCode,
					"jobs_file", jobsFile,
					"files", len(virtualDubFiles),
					"error", vdubErr,
					"process_output", virtualDubResult.Output,
				)
				for _, file := range virtualDubFiles {
					setFailure(file.result, stageErr)
				}
			} else {
				deshakeLogger.Debug("VirtualDub finished", "duration", duration, "exit_code", virtualDubResult.ExitCode)
				for _, file := range virtualDubFiles {
					size, err := verifyOutput(file.virtualDubOutput)
					if err != nil {
						failFile(baseLogger, runID, file.result, atStage(stageFileWritten, err))
						continue
					}
					file.current = media.Artifact{Path: file.virtualDubOutput, Type: media.ArtifactVideo, Media: file.input.Media}
					if file.ffmpegScale == nil {
						file.result.Status, file.result.Stage, file.result.Error = StatusCompleted, stageFileWritten, nil
						stageLogger(baseLogger, stageFileWritten, runID).Debug("final file written", "file", file.input.Path, "path", file.finalOutput, "size_bytes", size)
					} else {
						file.readyForFFmpeg = true
						deshakeLogger.Debug("VirtualDub output verified", "file", file.input.Path, "path", file.virtualDubOutput, "size_bytes", size)
					}
				}
			}
		}
	}

	scaleLogger := stageLogger(baseLogger, stageScale, runID)
	ffmpegRunner := ffmpegengine.Runner{Executable: r.Config.Tools.FFmpeg.Path, Logger: scaleLogger}
	for _, file := range prepared {
		if file.ffmpegScale == nil {
			continue
		}
		if !file.readyForFFmpeg {
			continue
		}
		if err := ctx.Err(); err != nil {
			file.result.Status, file.result.Stage, file.result.Error = StatusCancelled, stageScale, err
			continue
		}
		scaleLogger.Debug("scaling started",
			"file", file.input.Path,
			"input", file.current.Path,
			"output", file.finalOutput,
			"width", file.ffmpegScale.Config.Width,
			"height", file.ffmpegScale.Config.Height,
		)
		started := time.Now()
		ffmpegResult, err := ffmpegRunner.Run(ctx, file.current.Path, file.finalOutput, file.ffmpegScale.Config)
		if err != nil {
			failFile(baseLogger, runID, file.result, atStage(stageScale, err),
				"duration", time.Since(started),
				"exit_code", ffmpegResult.ExitCode,
				"process_output", ffmpegResult.Output,
			)
			continue
		}
		scaleLogger.Debug("scaling finished", "file", file.input.Path, "duration", time.Since(started), "exit_code", ffmpegResult.ExitCode)
		size, err := verifyOutput(file.finalOutput)
		if err != nil {
			failFile(baseLogger, runID, file.result, atStage(stageFileWritten, err))
			continue
		}
		if file.virtualDubOutput != "" {
			if err := os.Remove(file.virtualDubOutput); err != nil && !os.IsNotExist(err) {
				stageLogger(baseLogger, stageFinished, runID).Warn("HuffYUV intermediate cleanup failed", "file", file.input.Path, "path", file.virtualDubOutput, "error", err)
			}
		}
		file.result.Status, file.result.Stage, file.result.Error = StatusCompleted, stageFileWritten, nil
		stageLogger(baseLogger, stageFileWritten, runID).Debug("final file written", "file", file.input.Path, "path", file.finalOutput, "size_bytes", size)
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	return result, nil
}

func (r *Runner) planFile(ctx context.Context, filename string, result *FileResult, catalog *workflow.Catalog, store virtualdub.ConfigStore, logger *slog.Logger, runID string) (*plannedFile, error) {
	abs, err := filepath.Abs(filename)
	if err != nil {
		return nil, atStage(stagePreparation, err)
	}
	result.Input = abs
	info, err := accessibleFile(abs)
	if err != nil {
		return nil, atStage(stagePreparation, err)
	}
	stageLogger(logger, stagePreparation, runID).Debug("input file accessible", "file", abs, "size_bytes", info.Size())

	started := time.Now()
	probeLogger := stageLogger(logger, stageProbe, runID)
	spec, err := r.Probe.Probe(ctx, abs, probeLogger)
	if err != nil {
		return nil, atStage(stageProbe, err)
	}
	probeLogger.Debug("media attributes read",
		"file", abs,
		"duration", time.Since(started),
		"width", spec.Width,
		"height", spec.Height,
		"fps", spec.FPS,
		"scan", spec.ScanType,
		"field_order", spec.FieldOrder,
		"codec", spec.Codec,
		"pixel_format", spec.PixelFormat,
		"color_space", spec.ColorSpace,
		"audio_codec", spec.AudioCodec,
	)
	definition, err := catalog.Match(spec)
	if err != nil {
		return nil, atStage(stagePreparation, err)
	}
	stageLogger(logger, stagePreparation, runID).Debug("workflow selected",
		"file", abs,
		"workflow", definition.ID,
		"pipeline", activePipeline(definition),
	)
	file := &plannedFile{input: media.Artifact{Path: abs, Type: media.ArtifactSource, Media: spec}, result: result}
	lastEngineRank, activeFilters := -1, 0
	for _, node := range definition.Workflow {
		if !node.IsEnabled() {
			continue
		}
		activeFilters++
		instance, err := r.Registry.Create(node.ID, node.Filter, node.Config)
		if err != nil {
			return nil, atStage(stagePreparation, fmt.Errorf("workflow %q: %w", definition.ID, err))
		}
		rank := engineRank(instance.Engine())
		if rank < lastEngineRank {
			return nil, atStage(stagePreparation, fmt.Errorf("workflow %q has invalid engine order at filter %q", definition.ID, node.ID))
		}
		lastEngineRank = rank
		switch typed := instance.(type) {
		case *filter.AviSynthProfile:
			if file.profile != nil {
				return nil, atStage(stagePreparation, fmt.Errorf("workflow %q contains multiple AviSynth profiles", definition.ID))
			}
			if r.Config.Paths.AviSynthProfiles == "" {
				return nil, atStage(stagePreparation, fmt.Errorf("workflow %q uses avisynth.profile but paths.avisynth_profiles is empty", definition.ID))
			}
			file.profile = typed
		case *filter.Deshaker:
			if file.deshaker != nil {
				return nil, atStage(stagePreparation, fmt.Errorf("workflow %q contains multiple Deshaker filters", definition.ID))
			}
			if r.Config.Tools.VirtualDub.Executable == "" || r.Config.Paths.Deshaker == "" || r.Config.Paths.VirtualDubCodecs == "" {
				return nil, atStage(stagePreparation, fmt.Errorf("workflow %q uses virtualdub.deshaker but its tool or config paths are empty", definition.ID))
			}
			file.deshaker = typed
		case *filter.ZScale:
			if file.ffmpegScale != nil {
				return nil, atStage(stagePreparation, fmt.Errorf("workflow %q contains multiple ffmpeg.zscale filters", definition.ID))
			}
			if r.Config.Tools.FFmpeg.Path == "" {
				return nil, atStage(stagePreparation, fmt.Errorf("workflow %q uses ffmpeg.zscale but tools.ffmpeg.path is empty", definition.ID))
			}
			file.ffmpegScale = typed
		default:
			return nil, atStage(stagePreparation, fmt.Errorf("workflow %q contains unsupported filter %q", definition.ID, instance.Type()))
		}
	}
	if activeFilters == 0 {
		return nil, atStage(stagePreparation, fmt.Errorf("workflow %q contains no active filters", definition.ID))
	}
	if file.deshaker == nil && file.ffmpegScale == nil {
		return nil, atStage(stagePreparation, fmt.Errorf("workflow %q has no final media writer", definition.ID))
	}
	if file.deshaker != nil {
		file.deshakerTemplate, err = store.LoadDeshaker()
		if err != nil {
			return nil, atStage(stagePreparation, err)
		}
		codecID := "prores-pcm-mov"
		if file.ffmpegScale != nil {
			codecID = "huffyuv"
		}
		file.codec, err = store.LoadCodec(codecID)
		if err != nil {
			return nil, atStage(stagePreparation, err)
		}
	}
	base := strings.TrimSuffix(filepath.Base(abs), filepath.Ext(abs)) + "_processed"
	extension := ".mov"
	if file.ffmpegScale != nil {
		extension = file.ffmpegScale.Config.Extension
	} else if file.codec != nil {
		extension = file.codec.Extension
	}
	file.finalOutput = filepath.Join(r.Config.Processing.OutputDir, base+extension)
	result.Output = file.finalOutput
	return file, nil
}

func engineRank(engineType engine.Type) int {
	switch engineType {
	case engine.AviSynth:
		return 0
	case engine.VirtualDub:
		return 1
	case engine.FFmpeg:
		return 2
	default:
		return 100
	}
}

type stageError struct {
	stage string
	err   error
}

func (e *stageError) Error() string {
	return e.err.Error()
}

func (e *stageError) Unwrap() error {
	return e.err
}

func atStage(stage string, err error) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(*stageError); ok {
		return err
	}
	return &stageError{stage: stage, err: err}
}

func errorStage(err error) string {
	if staged, ok := err.(*stageError); ok {
		return staged.stage
	}
	return stagePreparation
}

func stageLogger(logger *slog.Logger, stage, runID string) *slog.Logger {
	return logger.With("stage", stage, "run_id", runID)
}

func accessibleFile(filename string) (os.FileInfo, error) {
	info, err := os.Stat(filename)
	if err != nil {
		return nil, fmt.Errorf("access input file %q: %w", filename, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("input is not a regular file: %s", filename)
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open input file %q: %w", filename, err)
	}
	if err := file.Close(); err != nil {
		return nil, fmt.Errorf("close input file %q: %w", filename, err)
	}
	return info, nil
}

func activePipeline(definition *workflow.Definition) string {
	stages := make([]string, 0, len(definition.Workflow))
	for _, node := range definition.Workflow {
		if node.IsEnabled() {
			stages = append(stages, node.Filter)
		}
	}
	return strings.Join(stages, " -> ")
}

func verifyOutput(filename string) (int64, error) {
	info, err := os.Stat(filename)
	if err != nil {
		return 0, fmt.Errorf("output was not created %q: %w", filename, err)
	}
	if info.Size() == 0 {
		return 0, fmt.Errorf("output is empty: %s", filename)
	}
	return info.Size(), nil
}

func skipFile(logger *slog.Logger, runID string, result *FileResult, err error) {
	result.Status, result.Stage, result.Error = StatusSkipped, errorStage(err), err
	stageLogger(logger, result.Stage, runID).Error("file skipped", "file", result.Input, "error", err)
}

func failFile(logger *slog.Logger, runID string, result *FileResult, err error, details ...any) {
	setFailure(result, err)
	attributes := []any{"file", result.Input, "error", err}
	attributes = append(attributes, details...)
	stageLogger(logger, result.Stage, runID).Error("file processing failed", attributes...)
}

func setFailure(result *FileResult, err error) {
	result.Status, result.Stage, result.Error = StatusFailed, errorStage(err), err
}

func markRemainingCancelled(result *BatchResult, start int, err error) {
	for i := start; i < len(result.Files); i++ {
		result.Files[i].Status, result.Files[i].Stage, result.Files[i].Error = StatusCancelled, stagePreparation, err
	}
}

func batchHasErrors(result *BatchResult) bool {
	for i := range result.Files {
		if result.Files[i].Error != nil {
			return true
		}
	}
	return false
}

type resultSummary struct {
	completed int
	skipped   int
	failed    int
	cancelled int
}

func summarize(result *BatchResult) resultSummary {
	var summary resultSummary
	for i := range result.Files {
		switch result.Files[i].Status {
		case StatusCompleted:
			summary.completed++
		case StatusSkipped:
			summary.skipped++
		case StatusFailed:
			summary.failed++
		case StatusCancelled:
			summary.cancelled++
		}
	}
	return summary
}

func finalResult(summary resultSummary, runErr error) string {
	if runErr != nil || summary.failed > 0 {
		return "failed"
	}
	if summary.cancelled > 0 {
		return "cancelled"
	}
	if summary.skipped > 0 {
		return "completed_with_skips"
	}
	return "completed"
}

func indexedName(index int, path, extension string) string {
	return fmt.Sprintf("%04d-%s%s", index, safeBase(path), extension)
}

func safeBase(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, base)
}
