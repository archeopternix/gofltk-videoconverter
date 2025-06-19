package ui

import (
	"log/slog"

	"github.com/pwiecz/go-fltk"
)

type ProjectConfig struct {
	OutputDir string // for encoded media files
	WorkDir   string // for intermediate files
	Encoder   string
	Cleanup   bool
}

func NewProjectConfig() ProjectConfig {
	return ProjectConfig{
		OutputDir: ".",
		WorkDir:   ".",
		Encoder:   "MP4 (x264 8bit)",
		Cleanup:   false,
	}
}

// var PrjCfg *ProjectConfig

// vdubConfigDialog creates and displays a modal dialog window
// to edit path to VirtualDub2, working and output directory and the used encoder.
func (p *ProjectConfig) Dialog() {
	// Create a modal window
	dialog := fltk.NewWindow(600, 260, "Project Configuration")
	dialog.SetModal() // Set the window as modal
	dialog.Begin()

	cfg := ProjectConfig{
		OutputDir: p.OutputDir,
		WorkDir:   p.WorkDir,
		Encoder:   p.Encoder,
		Cleanup:   p.Cleanup,
	}

	// Create a vertical box for layout
	mainBox := fltk.NewGroup(0, 0, dialog.W(), dialog.H())
	dialog.Add(mainBox)

	// Working Directory Button and Box
	workDirBox := fltk.NewBox(fltk.NO_BOX, 150, 10, 400, 30, "")
	if cfg.WorkDir == "" {
		cfg.WorkDir = "."
		workDirBox.SetLabel("No directory selected")
	} else {
		workDirBox.SetLabel(cfg.WorkDir)
	}
	workDirBox.SetAlign(fltk.ALIGN_LEFT | fltk.ALIGN_INSIDE) // Align text left and inside the box
	workDirBtn := fltk.NewButton(10, 10, 120, 30, "Working Directory")
	workDirBtn.SetCallback(func() {
		chooser := fltk.NewFileChooser(
			cfg.WorkDir,
			"*.*",
			fltk.FileChooser_DIRECTORY,
			"Choose Working Directory")
		chooser.Show()

		// Wait for user selection
		for chooser.Shown() {
			fltk.Wait()
		}
		if len(chooser.Selection()) > 0 {
			workDirBox.SetLabel(chooser.Selection()[0])
			cfg.WorkDir = chooser.Selection()[0]
		}
	})

	// Output Directory Button and Box
	outDirBox := fltk.NewBox(fltk.NO_BOX, 150, 50, 400, 30, "No directory selected")
	if cfg.OutputDir == "" {
		cfg.OutputDir = "."
		outDirBox.SetLabel("No directory selected")
	} else {
		outDirBox.SetLabel(cfg.OutputDir)
	}
	outDirBox.SetAlign(fltk.ALIGN_LEFT | fltk.ALIGN_INSIDE) // Align text left and inside the box
	outDirBtn := fltk.NewButton(10, 50, 120, 30, "Output Directory")
	outDirBtn.SetCallback(func() {
		chooser := fltk.NewFileChooser(
			cfg.OutputDir,
			"*.*",
			fltk.FileChooser_DIRECTORY,
			"Choose Output Directory")
		chooser.Show()

		// Wait for user selection
		for chooser.Shown() {
			fltk.Wait()
		}

		if len(chooser.Selection()) > 0 {
			outDirBox.SetLabel(chooser.Selection()[0])
			cfg.OutputDir = chooser.Selection()[0]
		}
	})

	// Encoder Dropdown
	encoderBox := fltk.NewBox(fltk.NO_BOX, 10, 90, 120, 30, "Encoder for video")
	encoderBox.SetAlign(fltk.ALIGN_LEFT | fltk.ALIGN_INSIDE) // Align text left and inside the box

	encoderChoice := fltk.NewChoice(150, 90, 200, 30, "")
	encoderChoice.Add("MP4 (x264 8bit)", func() {
		cfg.Encoder = "MP4 (x264 8bit)"
	})
	encoderChoice.Add("MP4 (x264 10bit)", func() {
		cfg.Encoder = "MP4 (x264 10bit)"
	})
	encoderChoice.Add("Huffyuv (lossless)", func() {
		cfg.Encoder = "Huffyuv (lossless)"
	})
	encoderChoice.Add("MP4 (x265 HEVC)", func() {
		cfg.Encoder = "MP4 (x265 HEVC)"
	})
	index := encoderChoice.FindIndex(cfg.Encoder)
	encoderChoice.SetValue(index)
	mainBox.Add(encoderChoice)

	// Cleanup Checkbox
	cbBox := fltk.NewBox(fltk.NO_BOX, 10, 130, 120, 30, "Clean-up files?")
	cbBox.SetAlign(fltk.ALIGN_LEFT | fltk.ALIGN_INSIDE) // Align text left and inside the box
	cb := fltk.NewCheckButton(150, 130, 20, 20, "")
	cb.SetValue(cfg.Cleanup)

	mainBox.Add(workDirBtn)
	mainBox.Add(workDirBox)
	mainBox.Add(outDirBtn)
	mainBox.Add(outDirBox)
	mainBox.Add(cbBox)
	mainBox.Add(cb)

	// Bottom Buttons
	bottomGroup := fltk.NewGroup(0, mainBox.H()-55, mainBox.W()-10, 40)

	cancelBtn := fltk.NewButton(bottomGroup.W()/2-110, mainBox.H()-40, 100, 30, "Cancel")
	saveBtn := fltk.NewButton(bottomGroup.W()/2+10, mainBox.H()-40, 100, 30, "Save")
	cancelBtn.SetCallback(func() {
		slog.Debug("project config discarded")
		dialog.Hide()
	})
	saveBtn.SetCallback(func() {
		slog.Debug("project config changed", "config", cfg)
		p.Cleanup = cfg.Cleanup
		p.Encoder = cfg.Encoder
		p.OutputDir = cfg.OutputDir
		p.WorkDir = cfg.WorkDir
		dialog.Hide()
	})
	bottomGroup.Add(cancelBtn)
	bottomGroup.Add(saveBtn)
	mainBox.Add(bottomGroup)

	// Finalize the window and display it
	dialog.End()
	dialog.Show()
}
