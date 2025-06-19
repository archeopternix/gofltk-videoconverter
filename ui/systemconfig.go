package ui

import (
	"log/slog"

	"github.com/pwiecz/go-fltk"
)

type SystemConfig struct {
	AvisynthPlugInPath string // path to AviSynth plugins
	VirtualDubPath     string // path to VirtualDub
}

func NewSystemConfig(avis, vdub string) SystemConfig {
	return SystemConfig{AvisynthPlugInPath: avis, VirtualDubPath: vdub}
}

//var SysCfg *SystemConfig

// vdubConfigDialog creates and displays a modal dialog window
// to edit path to VirtualDub2, working and output directory and the used encoder.
func (s *SystemConfig) Dialog() {
	// Create a modal window
	dialog := fltk.NewWindow(600, 220, "System Configuration")
	dialog.SetModal() // Set the window as modal
	dialog.Begin()

	// temp structure to revert when hit the cancel button
	cfg := SystemConfig{
		AvisynthPlugInPath: s.AvisynthPlugInPath,
		VirtualDubPath:     s.VirtualDubPath,
	}

	// Create a vertical box for layout
	mainBox := fltk.NewGroup(0, 0, dialog.W(), dialog.H())
	dialog.Add(mainBox)

	// Working Directory Button and Box
	avsDirBox := fltk.NewBox(fltk.NO_BOX, 10, 10, 400, 30, "")
	if cfg.AvisynthPlugInPath == "" {
		avsDirBox.SetLabel("No directory selected")
	} else {
		avsDirBox.SetLabel(cfg.AvisynthPlugInPath)
	}
	avsDirBox.SetAlign(fltk.ALIGN_LEFT | fltk.ALIGN_INSIDE) // Align text left and inside the box
	avsDirBtn := fltk.NewButton(410, 10, 140, 30, "Avisynth+ plugins")
	avsDirBtn.SetCallback(func() {
		chooser := fltk.NewFileChooser(
			cfg.AvisynthPlugInPath,
			"*.*",
			fltk.FileChooser_DIRECTORY,
			"Choose Avisynth+ Plugins Directory")
		chooser.Show()

		// Wait for user selection
		for chooser.Shown() {
			fltk.Wait()
		}
		if len(chooser.Selection()) > 0 {
			avsDirBox.SetLabel(chooser.Selection()[0])
			cfg.AvisynthPlugInPath = chooser.Selection()[0]
		}
	})

	// Path to VirtualDub2
	vdubDirBox := fltk.NewBox(fltk.NO_BOX, 10, 50, 400, 30, "")
	if cfg.VirtualDubPath == "" {
		vdubDirBox.SetLabel("No directory selected")
	} else {
		vdubDirBox.SetLabel(cfg.VirtualDubPath)
	}

	vdubDirBox.SetAlign(fltk.ALIGN_LEFT | fltk.ALIGN_INSIDE) // Align text left and inside the box
	vdubDirBtn := fltk.NewButton(410, 50, 140, 30, "VirtualDub2 path")
	vdubDirBtn.SetCallback(func() {
		chooser := fltk.NewFileChooser(
			cfg.VirtualDubPath,
			"VirtualDub*.*",
			fltk.FileChooser_SINGLE,
			"Choose VirtualDub2 executable")
		chooser.Show()

		// Wait for user selection
		for chooser.Shown() {
			fltk.Wait()
		}
		if len(chooser.Selection()) > 0 {
			vdubDirBox.SetLabel(chooser.Selection()[0])
			cfg.VirtualDubPath = chooser.Selection()[0]
		}
	})

	mainBox.Add(avsDirBtn)
	mainBox.Add(avsDirBox)
	mainBox.Add(vdubDirBtn)
	mainBox.Add(vdubDirBox)

	// Bottom Buttons
	bottomGroup := fltk.NewGroup(0, mainBox.H()-55, mainBox.W()-10, 40)

	cancelBtn := fltk.NewButton(bottomGroup.W()/2-110, mainBox.H()-40, 100, 30, "Cancel")
	saveBtn := fltk.NewButton(bottomGroup.W()/2+10, mainBox.H()-40, 100, 30, "Save")
	cancelBtn.SetCallback(func() {
		slog.Debug("System config discarded")
		dialog.Hide()
	})
	saveBtn.SetCallback(func() {
		slog.Debug("System config changed", "config", cfg)
		s.AvisynthPlugInPath = cfg.AvisynthPlugInPath
		s.VirtualDubPath = cfg.VirtualDubPath
		dialog.Hide()
	})
	bottomGroup.Add(cancelBtn)
	bottomGroup.Add(saveBtn)
	mainBox.Add(bottomGroup)

	// Finalize the window and display it
	dialog.End()
	dialog.Show()
}
