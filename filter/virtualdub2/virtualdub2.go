package virtualdub2

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"text/template"

	. "github.com/archeopternix/gofltk-videoconverter/filter"
	"github.com/archeopternix/gofltk-videoconverter/util"
)

// write a batch file in the folder of VirtualDub64.exe
// Content:
//
//	"C:\Program Files\VirtualDub64\VirtualDub64.exe" /s"%1" /x
const vDubWinPath = "C:\\\\Programme\\VirtualDub64\\exec.bat"
const vDubLinuxPath = "/home/archeopternix/VirtualDub2/VirtualDub64.exe"

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

	if v.OutDir == "" {
		v.OutDir = util.GetPathFromFile(files[0])
	}

	v.data.JobsTotal = len(files) * 2

	for i, file := range files {
		v.InFiles = append(v.InFiles, file)
		out := util.ReplacePathOfFile(file, v.OutDir)
		v.OutFiles = append(v.InFiles, util.ReplaceExtOfFile(out, v.Ext))
		v.Optionals = append(v.Optionals, util.ReplaceExtOfFile(out, ".log"))

		d1 := DataJob{
			Index:   i*2 + 1,
			InFile:  DoubleBackslash(file),
			OutFile: "",
			LogFile: DoubleBackslash(util.ReplaceExtOfFile(out, ".log")),
			Deshake: 1,
		}
		v.data.Jobs = append(v.data.Jobs, d1)

		d2 := DataJob{
			Index:   i*2 + 2,
			InFile:  DoubleBackslash(file),
			OutFile: DoubleBackslash(util.ReplaceExtOfFile(out, v.Ext)),
			LogFile: DoubleBackslash(util.ReplaceExtOfFile(out, ".log")),
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

func DoubleBackslash(in string) string {
	return strings.Replace(in, "\\", "\\\\", -1)
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
	out := util.ReplacePathOfFile("vdub.vcf", v.OutDir)
	f, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("could not create file '%s', %w", out, err)
	}
	defer f.Close()

	// execute template with data and write to f
	if err := tpl.ExecuteTemplate(f, "main", v.data); err != nil {
		return fmt.Errorf("error executing template: %w", err)
	}

	// Command to run VirtualDub with the batch job file
	cmd := exec.Command(vDubWinPath, out)

	slog.Debug("starting VirtualDub2 process", "cmd", cmd)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%v, %s", err, output)
	}
	slog.Debug("finished VirtualDub2.exe", "jobs:", out)

	return nil
}
