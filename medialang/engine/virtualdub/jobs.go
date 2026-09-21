package virtualdub

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

type Job struct {
	FileIndex  int
	InputPath  string
	OutputPath string
	LogPath    string
	Deshaker   *DeshakerTemplate
	Codec      *CodecPreset
}

type JobsBuilder struct {
	Pather      *PathConverter
	TemplateDir string
}

type headerData struct {
	Date     string
	Jobcount int
}

type jobData struct {
	Num             int
	FileIndex       int
	InFile          string
	OutFile         string
	LogFile         string
	ContainerScript string
	VideoScript     string
	AudioScript     string
	AnalysisScript  string
	RenderScript    string
	SaveScript      string
}

func (b JobsBuilder) Write(filename string, jobs []Job) (int, error) {
	tpl, err := template.New("virtualdub").Option("missingkey=error").ParseFiles(
		filepath.Join(b.TemplateDir, "header.tpl"),
		filepath.Join(b.TemplateDir, "analysis.tpl"),
		filepath.Join(b.TemplateDir, "render.tpl"),
	)
	if err != nil {
		return 0, fmt.Errorf("parse VirtualDub templates: %w", err)
	}

	var output bytes.Buffer
	jobCount := len(jobs) * 2
	if err := tpl.ExecuteTemplate(&output, "Header", headerData{
		Date:     time.Now().Format(time.RFC3339),
		Jobcount: jobCount,
	}); err != nil {
		return 0, fmt.Errorf("render VirtualDub header: %w", err)
	}

	num := 1
	for _, job := range jobs {
		data, err := b.resolve(job)
		if err != nil {
			return 0, err
		}
		data.Num = num
		if err := tpl.ExecuteTemplate(&output, "VDubFirst", data); err != nil {
			return 0, fmt.Errorf("render VirtualDub analysis job %d: %w", num, err)
		}
		num++
		data.Num = num
		if err := tpl.ExecuteTemplate(&output, "VDubSecond", data); err != nil {
			return 0, fmt.Errorf("render VirtualDub render job %d: %w", num, err)
		}
		num++
	}
	output.WriteString("//\n//--------------------------------------------------\n// $done\n")

	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return 0, fmt.Errorf("create VirtualDub work directory: %w", err)
	}
	if err := os.WriteFile(filename, output.Bytes(), 0o644); err != nil {
		return 0, fmt.Errorf("write VirtualDub jobs file %q: %w", filename, err)
	}
	return jobCount, nil
}

func (b JobsBuilder) resolve(job Job) (jobData, error) {
	inFile, err := b.Pather.ToWindowsPath(job.InputPath)
	if err != nil {
		return jobData{}, err
	}
	outFile, err := b.Pather.ToWindowsPath(job.OutputPath)
	if err != nil {
		return jobData{}, err
	}
	logFile, err := b.Pather.ToWindowsPath(job.LogPath)
	if err != nil {
		return jobData{}, err
	}
	values := map[string]any{
		"FileIndex": job.FileIndex,
		"InFile":    escapeSylia(inFile),
		"OutFile":   escapeSylia(outFile),
		"LogFile":   escapeSylia(logFile),
	}
	containerScript, err := renderFragment("container", job.Codec.ContainerScript, values)
	if err != nil {
		return jobData{}, err
	}
	videoScript, err := renderFragment("video", job.Codec.VideoScript, values)
	if err != nil {
		return jobData{}, err
	}
	audioScript, err := renderFragment("audio", job.Codec.AudioScript, values)
	if err != nil {
		return jobData{}, err
	}
	analysisScript, err := renderFragment("analysis", job.Deshaker.AnalysisScript, values)
	if err != nil {
		return jobData{}, err
	}
	renderScript, err := renderFragment("render", job.Deshaker.RenderScript, values)
	if err != nil {
		return jobData{}, err
	}
	saveScript, err := renderFragment("save", job.Codec.SaveScript, values)
	if err != nil {
		return jobData{}, err
	}
	return jobData{
		FileIndex:       job.FileIndex,
		InFile:          escapeSylia(inFile),
		OutFile:         escapeSylia(outFile),
		LogFile:         escapeSylia(logFile),
		ContainerScript: containerScript,
		VideoScript:     videoScript,
		AudioScript:     audioScript,
		AnalysisScript:  analysisScript,
		RenderScript:    renderScript,
		SaveScript:      saveScript,
	}, nil
}

func renderFragment(name, source string, values map[string]any) (string, error) {
	tpl, err := template.New(name).Option("missingkey=error").Parse(source)
	if err != nil {
		return "", fmt.Errorf("parse VirtualDub %s fragment: %w", name, err)
	}
	var output bytes.Buffer
	if err := tpl.Execute(&output, values); err != nil {
		return "", fmt.Errorf("render VirtualDub %s fragment: %w", name, err)
	}
	return output.String(), nil
}

func escapeSylia(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	return strings.ReplaceAll(value, `"`, `\"`)
}
