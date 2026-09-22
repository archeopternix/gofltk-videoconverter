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

type FileResult struct {
	Input    string
	Workflow string
	Output   string
	Stage    string
	Status   FileStatus
	Error    error
}

type BatchResult struct {
	RunID    string
	JobsFile string
	Files    []FileResult
}

type Runner struct {
	Config   *config.App
	Files    []string
	Probe    probe.Probe
	Registry *filter.Registry
	Logger   *slog.Logger
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

func NewRunner(app *config.App, files []string) *Runner {
	return &Runner{Config: app, Files: append([]string(nil), files...), Probe: probe.FFProbe{Executable: app.Tools.FFprobe.Path}, Registry: filter.DefaultRegistry(), Logger: slog.Default()}
}

func (r *Runner) Run(ctx context.Context) (*BatchResult, error) {
	runID := time.Now().Format("20060102T150405.000")
	baseLogger := r.Logger
	if baseLogger == nil {
		baseLogger = slog.Default()
	}
	logger := baseLogger.With("run_id", runID)
	result := &BatchResult{RunID: runID, Files: make([]FileResult, len(r.Files))}
	for i, filename := range r.Files {
		result.Files[i] = FileResult{Input: filename, Status: StatusSkipped}
	}

	catalog, err := workflow.LoadCatalog(r.Config.Paths.Workflows)
	if err != nil {
		return result, err
	}
	store := virtualdub.ConfigStore{CodecsDir: r.Config.Paths.VirtualDubCodecs, DeshakerFile: r.Config.Paths.Deshaker}
	pather := virtualdub.NewPathConverter(r.Config.Tools.VirtualDub)
	workDir := filepath.Join(r.Config.Processing.WorkDir, runID)
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return result, fmt.Errorf("create run directory %q: %w", workDir, err)
	}
	runCompleted := false
	defer func() {
		if r.Config.Processing.KeepFiles {
			logger.Info("work directory preserved", "stage", "cleanup", "status", "skipped", "directory", workDir, "reason", "keep_temp_files is enabled")
			return
		}
		if !runCompleted || batchHasErrors(result) {
			logger.Info("work directory preserved", "stage", "cleanup", "status", "skipped", "directory", workDir, "reason", "run did not complete successfully")
			return
		}
		if err := os.RemoveAll(workDir); err != nil {
			logger.Warn("work directory cleanup failed", "stage", "cleanup", "status", "failed", "directory", workDir, "error", err)
			return
		}
		logger.Info("work directory removed", "stage", "cleanup", "status", "completed", "directory", workDir)
	}()
	logger.Info("run started", "stage", "run", "status", "started", "files", len(r.Files), "work_directory", workDir)
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
		file, err := r.planFile(ctx, filename, &result.Files[i], catalog, store, logger)
		if err != nil {
			r.skip(logger, &result.Files[i], err)
			continue
		}
		key := strings.ToLower(filepath.Clean(file.finalOutput))
		if _, exists := outputs[key]; exists {
			r.skip(logger, &result.Files[i], atStage("workflow", fmt.Errorf("output path is already used in this run: %s", file.finalOutput)))
			continue
		}
		outputs[key] = struct{}{}
		planned = append(planned, file)
	}

	avsCompiler := avisynth.Compiler{ProfilesDir: r.Config.Paths.AviSynthProfiles, AviSynthPath: r.Config.Paths.AviSynth, Pather: pather}
	prepared := make([]*plannedFile, 0, len(planned))
	for _, file := range planned {
		if err := ctx.Err(); err != nil {
			file.result.Status, file.result.Stage, file.result.Error = StatusCancelled, "avisynth", err
			continue
		}
		file.fileIndex = len(prepared) + 1
		file.current = file.input
		if file.profile != nil {
			avsPath := filepath.Join(workDir, indexedName(file.fileIndex, file.input.Path, ".avs"))
			started := time.Now()
			artifact, err := avsCompiler.Compile(file.current, file.profile, avsPath)
			if err != nil {
				r.skip(logger, file.result, atStage("avisynth", err))
				continue
			}
			file.current = artifact
			logger.Info("AviSynth file written",
				"file", file.input.Path,
				"stage", "avisynth",
				"status", "completed",
				"path", avsPath,
				"duration", time.Since(started),
			)
		} else {
			logger.Debug("optional stage skipped", "file", file.input.Path, "stage", "avisynth", "status", "skipped", "reason", "not part of workflow")
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
			stageErr := atStage("jobs", err)
			logger.Error("VirtualDub jobs file failed", "stage", "jobs", "status", "failed", "path", jobsFile, "error", err)
			for _, file := range virtualDubFiles {
				r.setFailure(file.result, stageErr)
			}
		} else {
			logger.Info("VirtualDub jobs file written", "stage", "jobs", "status", "completed", "path", jobsFile, "jobs", jobCount, "files", len(virtualDubFiles))
			vdubRunner := virtualdub.Runner{Tool: r.Config.Tools.VirtualDub, Pather: pather, Logger: logger}
			logger.Info("VirtualDub started",
				"stage", "virtualdub",
				"status", "started",
				"executable", r.Config.Tools.VirtualDub.Executable,
				"jobs_file", jobsFile,
				"files", len(virtualDubFiles),
			)
			started := time.Now()
			virtualDubResult, runErr := vdubRunner.Run(ctx, jobsFile)
			duration := time.Since(started)
			if runErr != nil {
				stageErr := atStage("virtualdub", runErr)
				logger.Error("VirtualDub failed",
					"stage", "virtualdub",
					"status", "failed",
					"duration", duration,
					"exit_code", virtualDubResult.ExitCode,
					"jobs_file", jobsFile,
					"files", len(virtualDubFiles),
					"error", runErr,
					"process_output", virtualDubResult.Output,
				)
				for _, file := range virtualDubFiles {
					r.setFailure(file.result, stageErr)
				}
			} else {
				logger.Info("VirtualDub finished", "stage", "virtualdub", "status", "completed", "duration", duration, "exit_code", virtualDubResult.ExitCode)
				for _, file := range virtualDubFiles {
					size, err := verifyOutput(file.virtualDubOutput)
					if err != nil {
						r.fail(logger, file.result, atStage("output", err))
						continue
					}
					file.current = media.Artifact{Path: file.virtualDubOutput, Type: media.ArtifactVideo, Media: file.input.Media}
					if file.ffmpegScale == nil {
						file.result.Status, file.result.Stage, file.result.Error = StatusCompleted, "output", nil
						logger.Info("final file written", "file", file.input.Path, "stage", "output", "status", "completed", "path", file.finalOutput, "size_bytes", size)
						logger.Debug("optional stage skipped", "file", file.input.Path, "stage", "scaling", "status", "skipped", "reason", "not part of workflow")
					} else {
						file.readyForFFmpeg = true
						logger.Info("VirtualDub output verified", "file", file.input.Path, "stage", "virtualdub", "status", "completed", "path", file.virtualDubOutput, "size_bytes", size)
					}
				}
			}
		}
	}

	ffmpegRunner := ffmpegengine.Runner{Executable: r.Config.Tools.FFmpeg.Path, Logger: logger}
	for _, file := range prepared {
		if file.ffmpegScale == nil {
			if file.deshaker == nil {
				logger.Debug("optional stage skipped", "file", file.input.Path, "stage", "scaling", "status", "skipped", "reason", "not part of workflow")
			}
			continue
		}
		if !file.readyForFFmpeg {
			continue
		}
		if err := ctx.Err(); err != nil {
			file.result.Status, file.result.Stage, file.result.Error = StatusCancelled, "scaling", err
			continue
		}
		logger.Info("scaling started",
			"file", file.input.Path,
			"stage", "scaling",
			"status", "started",
			"input", file.current.Path,
			"output", file.finalOutput,
			"width", file.ffmpegScale.Config.Width,
			"height", file.ffmpegScale.Config.Height,
		)
		started := time.Now()
		ffmpegResult, err := ffmpegRunner.Run(ctx, file.current.Path, file.finalOutput, file.ffmpegScale.Config)
		if err != nil {
			r.failWithDetails(logger, file.result, atStage("scaling", err),
				"duration", time.Since(started),
				"exit_code", ffmpegResult.ExitCode,
				"process_output", ffmpegResult.Output,
			)
			continue
		}
		logger.Info("scaling finished", "file", file.input.Path, "stage", "scaling", "status", "completed", "duration", time.Since(started), "exit_code", ffmpegResult.ExitCode)
		size, err := verifyOutput(file.finalOutput)
		if err != nil {
			r.fail(logger, file.result, atStage("output", err))
			continue
		}
		if file.virtualDubOutput != "" {
			if err := os.Remove(file.virtualDubOutput); err != nil && !os.IsNotExist(err) {
				logger.Warn("HuffYUV intermediate cleanup failed", "file", file.input.Path, "stage", "cleanup", "status", "failed", "path", file.virtualDubOutput, "error", err)
			}
		}
		file.result.Status, file.result.Stage, file.result.Error = StatusCompleted, "output", nil
		logger.Info("final file written", "file", file.input.Path, "stage", "output", "status", "completed", "path", file.finalOutput, "size_bytes", size)
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	runCompleted = true
	logger.Info("run finished",
		"stage", "run",
		"status", runStatus(result),
		"completed", countStatus(result, StatusCompleted),
		"skipped", countStatus(result, StatusSkipped),
		"failed", countStatus(result, StatusFailed),
		"cancelled", countStatus(result, StatusCancelled),
	)
	return result, nil
}

func (r *Runner) planFile(ctx context.Context, filename string, result *FileResult, catalog *workflow.Catalog, store virtualdub.ConfigStore, logger *slog.Logger) (*plannedFile, error) {
	abs, err := filepath.Abs(filename)
	if err != nil {
		return nil, atStage("input", err)
	}
	result.Input = abs
	info, err := accessibleFile(abs)
	if err != nil {
		return nil, atStage("input", err)
	}
	logger.Info("input file accessible", "file", abs, "stage", "input", "status", "completed", "size_bytes", info.Size())

	started := time.Now()
	spec, err := r.Probe.Probe(ctx, abs)
	if err != nil {
		return nil, atStage("probe", err)
	}
	logger.Info("media attributes read",
		"file", abs,
		"stage", "probe",
		"status", "completed",
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
		return nil, atStage("workflow", err)
	}
	result.Workflow = definition.ID
	logger.Info("workflow selected",
		"file", abs,
		"stage", "workflow",
		"status", "selected",
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
			return nil, atStage("workflow", fmt.Errorf("workflow %q: %w", definition.ID, err))
		}
		rank := engineRank(instance.Engine())
		if rank < lastEngineRank {
			return nil, atStage("workflow", fmt.Errorf("workflow %q has invalid engine order at filter %q", definition.ID, node.ID))
		}
		lastEngineRank = rank
		switch typed := instance.(type) {
		case *filter.AviSynthProfile:
			if file.profile != nil {
				return nil, atStage("workflow", fmt.Errorf("workflow %q contains multiple AviSynth profiles", definition.ID))
			}
			if r.Config.Paths.AviSynthProfiles == "" {
				return nil, atStage("workflow", fmt.Errorf("workflow %q uses avisynth.profile but paths.avisynth_profiles is empty", definition.ID))
			}
			file.profile = typed
		case *filter.Deshaker:
			if file.deshaker != nil {
				return nil, atStage("workflow", fmt.Errorf("workflow %q contains multiple Deshaker filters", definition.ID))
			}
			if r.Config.Tools.VirtualDub.Executable == "" || r.Config.Paths.Deshaker == "" || r.Config.Paths.VirtualDubCodecs == "" {
				return nil, atStage("workflow", fmt.Errorf("workflow %q uses virtualdub.deshaker but its tool or config paths are empty", definition.ID))
			}
			file.deshaker = typed
		case *filter.ZScale:
			if file.ffmpegScale != nil {
				return nil, atStage("workflow", fmt.Errorf("workflow %q contains multiple ffmpeg.zscale filters", definition.ID))
			}
			if r.Config.Tools.FFmpeg.Path == "" {
				return nil, atStage("workflow", fmt.Errorf("workflow %q uses ffmpeg.zscale but tools.ffmpeg.path is empty", definition.ID))
			}
			file.ffmpegScale = typed
		default:
			return nil, atStage("workflow", fmt.Errorf("workflow %q contains unsupported filter %q", definition.ID, instance.Type()))
		}
	}
	if activeFilters == 0 {
		return nil, atStage("workflow", fmt.Errorf("workflow %q contains no active filters", definition.ID))
	}
	if file.deshaker == nil && file.ffmpegScale == nil {
		return nil, atStage("workflow", fmt.Errorf("workflow %q has no final media writer", definition.ID))
	}
	if file.deshaker != nil {
		file.deshakerTemplate, err = store.LoadDeshaker()
		if err != nil {
			return nil, atStage("workflow", err)
		}
		codecID := "prores-pcm-mov"
		if file.ffmpegScale != nil {
			codecID = "huffyuv"
		}
		file.codec, err = store.LoadCodec(codecID)
		if err != nil {
			return nil, atStage("workflow", err)
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
	logger.Debug("output planned", "file", abs, "stage", "workflow", "status", "completed", "output", file.finalOutput)
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
	return "unknown"
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

func (r *Runner) skip(logger *slog.Logger, result *FileResult, err error) {
	result.Status, result.Stage, result.Error = StatusSkipped, errorStage(err), err
	logger.Error("file skipped", "file", result.Input, "stage", result.Stage, "status", result.Status, "error", err)
}

func (r *Runner) fail(logger *slog.Logger, result *FileResult, err error) {
	r.failWithDetails(logger, result, err)
}

func (r *Runner) failWithDetails(logger *slog.Logger, result *FileResult, err error, details ...any) {
	r.setFailure(result, err)
	attributes := []any{"file", result.Input, "stage", result.Stage, "status", result.Status, "error", err}
	attributes = append(attributes, details...)
	logger.Error("file processing failed", attributes...)
}

func (r *Runner) setFailure(result *FileResult, err error) {
	result.Status, result.Stage, result.Error = StatusFailed, errorStage(err), err
}

func markRemainingCancelled(result *BatchResult, start int, err error) {
	for i := start; i < len(result.Files); i++ {
		result.Files[i].Status, result.Files[i].Stage, result.Files[i].Error = StatusCancelled, "run", err
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

func countStatus(result *BatchResult, status FileStatus) int {
	count := 0
	for i := range result.Files {
		if result.Files[i].Status == status {
			count++
		}
	}
	return count
}

func runStatus(result *BatchResult) string {
	if countStatus(result, StatusFailed) > 0 {
		return "failed"
	}
	if countStatus(result, StatusCancelled) > 0 {
		return "cancelled"
	}
	if countStatus(result, StatusSkipped) > 0 {
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
