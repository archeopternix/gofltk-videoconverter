package ui

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/archeopternix/gofltk-videoconverter/util"
	"github.com/pwiecz/go-fltk"
)

// App kapselt die Hauptlogik und UI-Elemente der Anwendung.
// Sie verwaltet das Hauptfenster, Menüleisten, Schaltflächen, Fortschrittsanzeige
// sowie die aktuelle Arbeitsumgebung.
//
// Felder:
//
//	win        – Das Hauptfenster der Anwendung.
//	MenuBar    – Die Menüleiste am oberen Rand des Fensters.
//	ButtonMenu – Ein Flex-Container für die Hauptaktionsschaltflächen (z.B. Öffnen, Ausführen).
//	Scroll     – (Nicht verwendet) Scroll-Widget, ggf. für Listenanzeige vorgesehen.
//	progress   – Fortschrittsbalken zur Anzeige des aktuellen Status (0–100%).
//	lister     – Benutzerdefiniertes Scroll-Widget zur Anzeige und Verwaltung von Dateieinträgen.
//	workDir    – Aktuelles Arbeitsverzeichnis für Dateioperationen.
type App struct {
	win           *fltk.Window   // Hauptfenster
	MenuBar       *fltk.MenuBar  // Menüleiste
	ButtonMenu    *fltk.Flex     // Container für Aktionsschaltflächen
	Scroll        *fltk.Scroll   // (Optional) Scroll-Widget
	progress      *fltk.Progress // Fortschrittsanzeige
	lister        *Scroll        // Benutzerdefinierte Liste für Dateien
	workDir       string         // Arbeitsverzeichnis
	sysconfig     SystemConfig
	projectconfig ProjectConfig
}

func NewApp(window *fltk.Window) *App {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}

	app := &App{
		win:           window,
		workDir:       wd,
		sysconfig:     NewSystemConfig(".", "."),
		projectconfig: NewProjectConfig(),
	}
	app.initMainWindow()
	return app
}

func (a *App) Exit() {
	a.win.Hide()
}

func (a App) Hello() {
	slog.Info("start")
}

// SetProgress sets the value 0..100% and a text message
func (a *App) SetProgress(num int, text string) {
	a.progress.SetValue(float64(num))
	a.progress.SetLabel(text)
}

const (
	labelSize = 11
)

func (a *App) initMainWindow() {
	a.win.Begin()
	// Button group
	a.ButtonMenu = fltk.NewFlex(0, 5, a.win.W(), 80)
	a.ButtonMenu.SetType(fltk.ROW)
	a.ButtonMenu.SetGap(1)
	a.ButtonMenu.Begin()

	openFileBtn := fltk.NewButton(10, 0, 80, 70, "Open File")
	openFileBtn.SetTooltip("Open File")
	openFileBtn.SetAlign(fltk.ALIGN_IMAGE_OVER_TEXT)
	imgFile, err := fltk.NewPngImageLoad("img/document-open.png")
	if err != nil {
		slog.Error("button open image", "image:", err)
	}
	openFileBtn.SetImage(imgFile)
	openFileBtn.SetCallback(func() {
		fmt.Println("OpenFile")
		a.openFile()
	})
	openFileBtn.SetLabelSize(labelSize)
	a.ButtonMenu.Fixed(openFileBtn, 80) // Fix width to 170 px

	openFolderBtn := fltk.NewButton(0, 0, 80, 70, "Open Folder")
	openFolderBtn.SetCallback(func() {
		fmt.Println("OpenFolderBtn")
	})
	openFolderBtn.SetAlign(fltk.ALIGN_IMAGE_OVER_TEXT)
	imgFolder, err := fltk.NewPngImageLoad("img/folder-open.png")
	if err != nil {
		slog.Error("button open folder", "image:", err)
	}
	openFolderBtn.SetImage(imgFolder)
	openFolderBtn.SetCallback(func() {
		fmt.Println("OpenFolder")
	})
	openFolderBtn.SetLabelSize(labelSize)
	a.ButtonMenu.Fixed(openFolderBtn, 80)

	sep1 := fltk.NewBox(fltk.NO_BOX, 0, 0, 20, 70, "")
	a.ButtonMenu.Fixed(sep1, 20)

	RunBtn := fltk.NewButton(0, 0, 80, 70, "Run")
	RunBtn.SetAlign(fltk.ALIGN_IMAGE_OVER_TEXT)
	imgRun, err := fltk.NewPngImageLoad("img/video-x-generic.png")
	if err != nil {
		slog.Error("button open folder", "image:", err)
	}
	RunBtn.SetLabelSize(labelSize)
	RunBtn.SetImage(imgRun)
	RunBtn.SetCallback(func() {
		fmt.Println("Run")
	})
	a.ButtonMenu.Fixed(RunBtn, 80) // Fix width to 170 px

	sep2 := fltk.NewBox(fltk.NO_BOX, 0, 0, 20, 70, "")
	a.ButtonMenu.Fixed(sep2, 20)

	SettingsBtn := fltk.NewButton(0, 0, 80, 70, "Settings")
	SettingsBtn.SetAlign(fltk.ALIGN_IMAGE_OVER_TEXT)
	imgSettings, err := fltk.NewPngImageLoad("img/applications-system.png")
	if err != nil {
		slog.Error("button open folder", "image:", err)
	}
	SettingsBtn.SetLabelSize(labelSize)
	SettingsBtn.SetImage(imgSettings)
	SettingsBtn.SetCallback(func() {
		fmt.Println("Settings")
		a.sysconfig.Dialog()
	})
	a.ButtonMenu.Fixed(SettingsBtn, 80) // Fix width to 170 px

	ConfigBtn := fltk.NewButton(0, 0, 80, 70, "Configuration")
	ConfigBtn.SetAlign(fltk.ALIGN_IMAGE_OVER_TEXT)
	imgConfig, err := fltk.NewPngImageLoad("img/preferences-desktop-theme.png")
	if err != nil {
		slog.Error("button open folder", "image:", err)
	}
	ConfigBtn.SetLabelSize(labelSize)
	ConfigBtn.SetImage(imgConfig)
	ConfigBtn.SetCallback(func() {
		fmt.Println("Config")
		a.projectconfig.Dialog()
	})
	a.ButtonMenu.Fixed(ConfigBtn, 80) // Fix width to 170 px

	bx := fltk.NewBox(fltk.NO_BOX, 0, 0, 1, 1, "")
	a.ButtonMenu.Add(bx)
	a.ButtonMenu.End()

	mainContent := fltk.NewFlex(0, 85, a.win.W(), a.win.H()-80-25)
	mainContent.Begin()
	a.lister = NewScroll(0, 0, mainContent.W(), mainContent.H())
	// ... add widgets to mainContent ...
	mainContent.End()

	a.win.Resizable(mainContent)

	a.progress = fltk.NewProgress(0, a.win.H()-25, a.win.W(), 25, "Status")
	// Set range (min, max)
	a.progress.SetMinimum(0)
	a.progress.SetSelectionColor(fltk.ColorFromRgb(180, 180, 180))
	a.progress.SetMaximum(100)
	// Set value (current progress)
	a.progress.SetValue(50)

	a.win.End()
}

/*
// ConfigWin sets up the main FLTK window with menu, scroll area, and buttons.
func MainWin(window *fltk.Window) {
	// Initialize filter lists
	filters := FilterList{}
	allfilters := filter.FilterNames

	// Set working directory to current if available, else "."
	w, err := os.Getwd()
	if err != nil {
		workingDir = "."
	} else {
		workingDir = w
	}

	win := window
	win.SetLabel("Zeilenliste mit FLTK") // Set window label

	win.Begin()

	// Create a menu bar at the top of the window
	menu := fltk.NewMenuBar(0, 0, win.W(), 25)

	// Add menu entries for file operations
	menu.Add("File/Open File...", openFile)
	menu.Add("File/Open Directory...", openDirectory)
	menu.Add("File/Generate Files", generateFiles)
	menu.Add("File/Exit", func() {
		os.Exit(0) // Exit the application
	})

	// Add menu entry for media filter configuration
	menu.Add("Media/Filters", func() {
		filterConfigDialog(&filters, allfilters)
	})

	// Add menu entry to delete selected media items
	menu.Add("Media/Delete selected", func() {
		for _, i := range lister.GetSelectedRows() {
			lister.DeleteRow(i)
		}
	})

	// Add menu entry for project settings dialog
	menu.Add("Options/Project Settings", func() {
		if PrjCfg == nil {
			PrjCfg = NewProjectConfig()
		}
		projectConfigDialog()
	})

	// Add menu entry for system settings dialog
	menu.Add("Options/System Settings", func() {
		if SysCfg == nil {
			SysCfg = &SystemConfig{}
		}
		systemConfigDialog()
	})

	// Create a scroll area for displaying file items
	scroll := NewScroll(0, 25, win.W()-5, 320)
	lister = scroll

	// Create a button group at the bottom of the window
	buttonGroup := fltk.NewGroup(0, 360, win.W(), 50)
	buttonGroup.Begin()

	// Add a button to delete selected rows from the list
	delBtn := fltk.NewButton(210, 360, 180, 40, "Ausgewählte löschen")
	delBtn.SetCallback(func() {
		for _, i := range lister.GetSelectedRows() {
			lister.DeleteRow(i)
		}
	})

	buttonGroup.End()

	// Make the scroll area resizable within the window
	win.Resizable(scroll.fltkScroll)

	win.End()
}


// openDirectory prompts the user to select a directory, processes its video files,
// and adds them to the scrollable list.
func openDirectory() {
	// Create a new directory chooser dialog
	chooser := fltk.NewFileChooser(
		workingDir,                 // Default directory
		"*.*",                      // File filter (all files)
		fltk.FileChooser_DIRECTORY, // Mode: Open directory
		"Select Directory",         // Dialog title
	)
	chooser.Show()

	// Wait for the user to make a selection
	for chooser.Shown() {
		fltk.Wait()
	}

	// Handle case where no directory was selected
	if len(chooser.Selection()) == 0 {
		slog.Info("open directory", "no files selected")
		return
	}

	// Set up a new workflow to process the directory contents
	wf := filter.NewWorkflow()
	selection := chooser.Selection()
	workingDir = selection[0]

	wf.Add(filter.NewDirectoryReader(selection[0])) // Add directory reader step
	wf.Add(filter.NewVideoFileFilter())             // Add video file filter step

	list, err := wf.Process(nil) // Process directory
	if err != nil {
		slog.Error("open directory", "message", err)
		return
	}

	// If no video files found, log and return
	if len(list) == 0 {
		slog.Info("open directory", "no video files selected")
		return
	}

	// Add each processed video file to the scrollable list
	for _, item := range list {
		lister.AddRow(util.GetInfoFromFileName(item))
	}
}

*/

// openFile prompts the user to select one or more video files, processes them,
// and adds them to the scrollable list.
func (a *App) openFile() {
	// Create a new file chooser dialog for video files
	chooser := fltk.NewFileChooser(
		a.workDir,                          // Default directory
		"*.{mp4,mpeg,avi,vob,mpg,mov,m2t}", // Video file filter
		fltk.FileChooser_MULTI,             // Mode: Select multiple files
		"Select File",                      // Dialog title
	)
	chooser.Show()

	// Wait for the user to make a selection
	for chooser.Shown() {
		fltk.Wait()
	}

	// Handle case where no files are selected
	if len(chooser.Selection()) == 0 {
		slog.Info("open directory", "no files selected")
		return
	}

	var videofiles []string

	for _, file := range chooser.Selection() {
		if util.IsVideo(file) {
			videofiles = append(videofiles, filepath.ToSlash(file))
		}
	}

	// If no valid video files found, log and return
	if len(videofiles) == 0 {
		slog.Info("open files", "no video files selected")
		return
	}

	// Update working directory to the location of the first file
	a.workDir, _ = filepath.Split(videofiles[0])
	// Add each processed video file to the scrollable list
	for _, item := range videofiles {
		a.lister.AddRow(util.GetInfoFromFileName(item))
	}
}

/*
// generateFiles collects all file paths from the scroll list
// and initializes a new workflow for further processing (e.g., batch processing).
func generateFiles() {
	// 1. Collect all file paths from the scroll list
	files := lister.GetFiles()
	slog.Debug("Files to process", "files", files)

	// 2. Create a new workflow using filter/workflow package
	//	wf := workflow.NewWorkflow() // Replace with correct constructor if needed

	// [Further processing steps would go here]
}
*/
