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
	index    int
	input    media.Artifact
	workflow *workflow.Definition
	profile  *filter.AviSynthProfile
	deshaker *filter.Deshaker
	codec    *virtualdub.CodecPreset
	output   string
	log      string
	current  media.Artifact
	result   *FileResult
}

func NewRunner(app *config.App, files []string) *Runner {
	return &Runner{
		Config:   app,
		Files:    append([]string(nil), files...),
		Probe:    probe.FFProbe{Executable: app.Tools.FFprobe.Path},
		Registry: filter.DefaultRegistry(),
		Logger:   slog.Default(),
	}
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
	store := virtualdub.ConfigStore{
		CodecsDir:    r.Config.Paths.VirtualDubCodecs,
		DeshakerFile: r.Config.Paths.Deshaker,
	}
	pather := virtualdub.NewPathConverter(r.Config.Tools.VirtualDub)
	workDir := filepath.Join(r.Config.Processing.WorkDir, runID)
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return result, fmt.Errorf("create run directory %q: %w", workDir, err)
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
		file, err := r.planFile(ctx, i, filename, &result.Files[i], catalog, store)
		if err != nil {
			r.skip(&result.Files[i], err)
			continue
		}
		key := strings.ToLower(filepath.Clean(file.output))
		if _, exists := outputs[key]; exists {
			r.skip(&result.Files[i], fmt.Errorf("output path is already used in this run: %s", file.output))
			continue
		}
		outputs[key] = struct{}{}
		planned = append(planned, file)
	}

	avsCompiler := avisynth.Compiler{ProfilesDir: r.Config.Paths.AviSynthProfiles, Pather: pather}
	jobs := make([]virtualdub.Job, 0, len(planned))
	active := make([]*plannedFile, 0, len(planned))
	for _, file := range planned {
		if err := ctx.Err(); err != nil {
			file.result.Status = StatusCancelled
			file.result.Error = err
			continue
		}
		file.current = file.input
		if file.profile != nil {
			avsPath := filepath.Join(workDir, "avisynth", fmt.Sprintf("%04d-%s.avs", file.index, safeBase(file.input.Path)))
			artifact, err := avsCompiler.Compile(file.current, file.profile, avsPath)
			if err != nil {
				r.skip(file.result, err)
				continue
			}
			file.current = artifact
		}

		var deshakerConfig *virtualdub.DeshakerTemplate
		if file.deshaker != nil {
			loaded, err := store.LoadDeshaker()
			if err != nil {
				r.skip(file.result, err)
				continue
			}
			deshakerConfig = loaded
		}
		job := virtualdub.Job{
			FileIndex:  file.index,
			InputPath:  file.current.Path,
			OutputPath: file.output,
			LogPath:    file.log,
			Deshaker:   deshakerConfig,
			Codec:      file.codec,
		}
		if file.deshaker != nil {
			job.ReuseAnalysis = file.deshaker.Config.ReuseAnalysis
			job.ForceAnalysis = file.deshaker.Config.ForceAnalysis
		}
		jobs = append(jobs, job)
		active = append(active, file)
	}

	if len(jobs) == 0 {
		return result, ctx.Err()
	}
	jobsFile := filepath.Join(workDir, "virtualdub", "medialang.jobs")
	result.JobsFile = jobsFile
	builder := virtualdub.JobsBuilder{Pather: pather}
	if _, err := builder.Write(jobsFile, jobs); err != nil {
		for _, file := range active {
			r.fail(file.result, err)
		}
		return result, err
	}

	vdubRunner := virtualdub.Runner{Tool: r.Config.Tools.VirtualDub, Pather: pather}
	if _, err := vdubRunner.Run(ctx, jobsFile); err != nil {
		status := StatusFailed
		if ctx.Err() != nil {
			status = StatusCancelled
		}
		for _, file := range active {
			file.result.Status = status
			file.result.Error = err
		}
		return result, err
	}

	for _, file := range active {
		info, err := os.Stat(file.output)
		if err != nil || info.Size() == 0 {
			if err == nil {
				err = fmt.Errorf("output is empty: %s", file.output)
			} else {
				err = fmt.Errorf("output was not created %q: %w", file.output, err)
			}
			r.fail(file.result, err)
			continue
		}
		file.result.Status = StatusCompleted
		file.result.Error = nil
	}
	return result, nil
}

func (r *Runner) planFile(ctx context.Context, index int, filename string, result *FileResult, catalog *workflow.Catalog, store virtualdub.ConfigStore) (*plannedFile, error) {
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
	result.Input = abs
	result.Workflow = definition.ID

	file := &plannedFile{
		index:    index,
		input:    media.Artifact{Path: abs, Type: media.ArtifactSource, Media: spec},
		workflow: definition,
		result:   result,
	}
	virtualDubSeen := false
	encodeSeen := false
	for _, node := range definition.Workflow {
		if !node.IsEnabled() {
			continue
		}
		instance, err := r.Registry.Create(node.ID, node.Filter, node.Config)
		if err != nil {
			return nil, fmt.Errorf("workflow %q: %w", definition.ID, err)
		}
		switch typed := instance.(type) {
		case *filter.AviSynthProfile:
			if virtualDubSeen || file.profile != nil {
				return nil, fmt.Errorf("workflow %q has an unsupported AviSynth position", definition.ID)
			}
			file.profile = typed
		case *filter.Deshaker:
			if encodeSeen || file.deshaker != nil {
				return nil, fmt.Errorf("workflow %q has an unsupported Deshaker position", definition.ID)
			}
			virtualDubSeen = true
			file.deshaker = typed
		case *filter.Encode:
			if encodeSeen {
				return nil, fmt.Errorf("workflow %q contains multiple encoders", definition.ID)
			}
			virtualDubSeen = true
			encodeSeen = true
			codec, err := store.LoadCodec(typed.Config.Preset)
			if err != nil {
				return nil, err
			}
			file.codec = codec
		default:
			return nil, fmt.Errorf("workflow %q contains unsupported filter %q", definition.ID, instance.Type())
		}
		if instance.Engine() != engine.AviSynth && instance.Engine() != engine.VirtualDub {
			return nil, fmt.Errorf("workflow %q contains unsupported engine %q", definition.ID, instance.Engine())
		}
	}
	if file.codec == nil {
		return nil, fmt.Errorf("workflow %q has no VirtualDub encoder", definition.ID)
	}

	base := strings.TrimSuffix(filepath.Base(abs), filepath.Ext(abs)) + "_processed"
	file.output = filepath.Join(r.Config.Processing.OutputDir, base+file.codec.Extension)
	file.log = filepath.Join(r.Config.Processing.OutputDir, base+".deshaker.log")
	result.Output = file.output
	return file, nil
}

func (r *Runner) skip(result *FileResult, err error) {
	result.Status = StatusSkipped
	result.Error = err
	r.Logger.Error("file skipped", "file", result.Input, "error", err)
}

func (r *Runner) fail(result *FileResult, err error) {
	result.Status = StatusFailed
	result.Error = err
	r.Logger.Error("file failed", "file", result.Input, "error", err)
}

func markRemainingCancelled(result *BatchResult, start int, err error) {
	for i := start; i < len(result.Files); i++ {
		result.Files[i].Status = StatusCancelled
		result.Files[i].Error = err
	}
}

func safeBase(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	base = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, base)
	return base
}
