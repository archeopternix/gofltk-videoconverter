package virtualdub2

import (
	"fmt"
	"os"
	"text/template"

	. "github.com/archeopternix/gofltk-videoconverter/filter"
	"github.com/archeopternix/gofltk-videoconverter/util"
)

type VDubFilter struct {
	Filter
	Ext  string // Extension can be .mp4 or _test.avi
	data Data
}

func NewVDubFilter(outExt string) *VDubFilter {
	f := &VDubFilter{
		Filter: *NewFilter("VDUB2"),
		Ext:    outExt,
	}
	return f
}

func (v *VDubFilter) AddFiles(files ...string) {
	if len(files) == 0 {
		return
	}

	outdir := util.GetPathFromFile(files[0])
	if v.OutDir != "" {
		outdir = v.OutDir
	}

	v.data.JobsTotal = len(files) * 2

	for i, file := range files {
		v.InFiles = append(v.InFiles, file)
		out := util.ReplacePathOfFile(file, outdir)
		v.OutFiles = append(v.InFiles, util.ReplaceExtOfFile(out, v.Ext))
		v.Optionals = append(v.Optionals, util.ReplaceExtOfFile(out, ".log"))

		d1 := DataJob{
			Index:   i*2 + 1,
			InFile:  file,
			OutFile: "",
			LogFile: util.ReplaceExtOfFile(out, ".log"),
			Deshake: 1,
		}
		v.data.Jobs = append(v.data.Jobs, d1)

		d2 := DataJob{
			Index:   i*2 + 2,
			InFile:  file,
			OutFile: util.ReplaceExtOfFile(out, v.Ext),
			LogFile: util.ReplaceExtOfFile(out, ".log"),
			Deshake: 2,
		}
		v.data.Jobs = append(v.data.Jobs, d2)
	}
}

type Data struct {
	JobsTotal int
	Jobs      []DataJob
}

type DataJob struct {
	Index   int
	InFile  string
	OutFile string
	LogFile string
	Deshake int // 1 = Analyze; 2 = Save
}

func (v VDubFilter) Run() error {
	if len(v.InFiles) == 0 {
		return fmt.Errorf("no files to process")
	}

	// open the template file
	tpl, err := template.ParseFiles("filter/virtualdub2/main.tpl")
	if err != nil {
		return fmt.Errorf("could not read template file '%s', %w", "main.tpl", err)
	}

	// write to vdub.jobs file
	out := util.ReplacePathOfFile("vdub.jobs", v.OutDir)
	f, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("could not create file '%s', %w", out, err)
	}
	defer f.Close()

	// execute template with data and write to f
	if err := tpl.ExecuteTemplate(f, "main", v.data); err != nil {
		return fmt.Errorf("error executing template: %w", err)
	}
	return nil
}
