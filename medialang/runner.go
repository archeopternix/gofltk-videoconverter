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
	if !r.Config.Processing.KeepFiles {
		defer func() {
			if err := os.RemoveAll(workDir); err != nil {
				r.Logger.Warn("cleanup work directory failed", "directory", workDir, "error", err)
			}
		}()
	}
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
		file, err := r.planFile(ctx, filename, &result.Files[i], catalog, store)
		if err != nil {
			r.skip(&result.Files[i], err)
			continue
		}
		key := strings.ToLower(filepath.Clean(file.finalOutput))
		if _, exists := outputs[key]; exists {
			r.skip(&result.Files[i], fmt.Errorf("output path is already used in this run: %s", file.finalOutput))
			continue
		}
		outputs[key] = struct{}{}
		planned = append(planned, file)
	}

	avsCompiler := avisynth.Compiler{ProfilesDir: r.Config.Paths.AviSynthProfiles, AviSynthPath: r.Config.Paths.AviSynth, Pather: pather}
	prepared := make([]*plannedFile, 0, len(planned))
	for _, file := range planned {
		if err := ctx.Err(); err != nil {
			file.result.Status, file.result.Error = StatusCancelled, err
			continue
		}
		file.fileIndex = len(prepared) + 1
		file.current = file.input
		if file.profile != nil {
			avsPath := filepath.Join(workDir, indexedName(file.fileIndex, file.input.Path, ".avs"))
			artifact, err := avsCompiler.Compile(file.current, file.profile, avsPath)
			if err != nil {
				r.skip(file.result, err)
				continue
			}
			file.current = artifact
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
		if _, err := builder.Write(jobsFile, jobs); err != nil {
			for _, file := range virtualDubFiles {
				r.fail(file.result, err)
			}
		} else {
			vdubRunner := virtualdub.Runner{Tool: r.Config.Tools.VirtualDub, Pather: pather}
			_, runErr := vdubRunner.Run(ctx, jobsFile)
			for _, file := range virtualDubFiles {
				if err := verifyOutput(file.virtualDubOutput); err != nil {
					if runErr != nil {
						err = fmt.Errorf("VirtualDub failed: %v; %w", runErr, err)
					}
					r.fail(file.result, err)
					continue
				}
				file.current = media.Artifact{Path: file.virtualDubOutput, Type: media.ArtifactVideo, Media: file.input.Media}
				if file.ffmpegScale == nil {
					file.result.Status, file.result.Error = StatusCompleted, nil
				} else {
					file.readyForFFmpeg = true
				}
			}
		}
	}

	ffmpegRunner := ffmpegengine.Runner{Executable: r.Config.Tools.FFmpeg.Path}
	for _, file := range prepared {
		if file.ffmpegScale == nil || !file.readyForFFmpeg {
			continue
		}
		if err := ctx.Err(); err != nil {
			file.result.Status, file.result.Error = StatusCancelled, err
			continue
		}
		if err := ffmpegRunner.Run(ctx, file.current.Path, file.finalOutput, file.ffmpegScale.Config); err != nil {
			r.fail(file.result, err)
			continue
		}
		if err := verifyOutput(file.finalOutput); err != nil {
			r.fail(file.result, err)
			continue
		}
		if file.virtualDubOutput != "" {
			if err := os.Remove(file.virtualDubOutput); err != nil && !os.IsNotExist(err) {
				r.Logger.Warn("cleanup HuffYUV intermediate failed", "file", file.virtualDubOutput, "error", err)
			}
		}
		file.result.Status, file.result.Error = StatusCompleted, nil
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	return result, nil
}

func (r *Runner) planFile(ctx context.Context, filename string, result *FileResult, catalog *workflow.Catalog, store virtualdub.ConfigStore) (*plannedFile, error) {
	abs, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}
	spec, err := r.Probe.Probe(ctx, abs)
	if err != nil {
		return nil, err
	}
	definition, err := catalog.Match(spec)
	if err != nil {
		return nil, err
	}
	result.Input, result.Workflow = abs, definition.ID
	file := &plannedFile{input: media.Artifact{Path: abs, Type: media.ArtifactSource, Media: spec}, result: result}
	lastEngineRank, activeFilters := -1, 0
	for _, node := range definition.Workflow {
		if !node.IsEnabled() {
			continue
		}
		activeFilters++
		instance, err := r.Registry.Create(node.ID, node.Filter, node.Config)
		if err != nil {
			return nil, fmt.Errorf("workflow %q: %w", definition.ID, err)
		}
		rank := engineRank(instance.Engine())
		if rank < lastEngineRank {
			return nil, fmt.Errorf("workflow %q has invalid engine order at filter %q", definition.ID, node.ID)
		}
		lastEngineRank = rank
		switch typed := instance.(type) {
		case *filter.AviSynthProfile:
			if file.profile != nil {
				return nil, fmt.Errorf("workflow %q contains multiple AviSynth profiles", definition.ID)
			}
			if r.Config.Paths.AviSynthProfiles == "" {
				return nil, fmt.Errorf("workflow %q uses avisynth.profile but paths.avisynth_profiles is empty", definition.ID)
			}
			file.profile = typed
		case *filter.Deshaker:
			if file.deshaker != nil {
				return nil, fmt.Errorf("workflow %q contains multiple Deshaker filters", definition.ID)
			}
			if r.Config.Tools.VirtualDub.Executable == "" || r.Config.Paths.Deshaker == "" || r.Config.Paths.VirtualDubCodecs == "" {
				return nil, fmt.Errorf("workflow %q uses virtualdub.deshaker but its tool or config paths are empty", definition.ID)
			}
			file.deshaker = typed
		case *filter.ZScale:
			if file.ffmpegScale != nil {
				return nil, fmt.Errorf("workflow %q contains multiple ffmpeg.zscale filters", definition.ID)
			}
			if r.Config.Tools.FFmpeg.Path == "" {
				return nil, fmt.Errorf("workflow %q uses ffmpeg.zscale but tools.ffmpeg.path is empty", definition.ID)
			}
			file.ffmpegScale = typed
		default:
			return nil, fmt.Errorf("workflow %q contains unsupported filter %q", definition.ID, instance.Type())
		}
	}
	if activeFilters == 0 {
		return nil, fmt.Errorf("workflow %q contains no active filters", definition.ID)
	}
	if file.deshaker == nil && file.ffmpegScale == nil {
		return nil, fmt.Errorf("workflow %q has no final media writer", definition.ID)
	}
	if file.deshaker != nil {
		file.deshakerTemplate, err = store.LoadDeshaker()
		if err != nil {
			return nil, err
		}
		codecID := "prores-pcm-mov"
		if file.ffmpegScale != nil {
			codecID = "huffyuv"
		}
		file.codec, err = store.LoadCodec(codecID)
		if err != nil {
			return nil, err
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

func verifyOutput(filename string) error {
	info, err := os.Stat(filename)
	if err != nil {
		return fmt.Errorf("output was not created %q: %w", filename, err)
	}
	if info.Size() == 0 {
		return fmt.Errorf("output is empty: %s", filename)
	}
	return nil
}

func (r *Runner) skip(result *FileResult, err error) {
	result.Status, result.Error = StatusSkipped, err
	r.Logger.Error("file skipped", "file", result.Input, "error", err)
}

func (r *Runner) fail(result *FileResult, err error) {
	result.Status, result.Error = StatusFailed, err
	r.Logger.Error("file failed", "file", result.Input, "error", err)
}

func markRemainingCancelled(result *BatchResult, start int, err error) {
	for i := start; i < len(result.Files); i++ {
		result.Files[i].Status, result.Files[i].Error = StatusCancelled, err
	}
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
