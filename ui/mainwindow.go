package ui

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/pwiecz/go-fltk"
)

/*	"log/slog"

	"path/filepath"
*/

type App struct {
	win        *fltk.Window
	MenuBar    *fltk.MenuBar
	ButtonMenu *fltk.Flex
	Scroll     *fltk.Scroll

	workDir string
}

func NewApp(window *fltk.Window) *App {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}

	app := &App{
		win:     window,
		workDir: wd,
	}
	app.initMainWindow()
	return app
}

func (a *App) Exit() {
	a.win.Hide()
}

func (a App) Hello() {
	slog.Info("hello")
}

func (a *App) initMainWindow() {
	a.win.Begin()
	// Button group
	a.ButtonMenu = fltk.NewFlex(0, 0, a.win.W(), 80)
	a.ButtonMenu.SetType(fltk.ROW)
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
	})
	openFileBtn.SetLabelSize(12)
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
	openFolderBtn.SetLabelSize(12)
	a.ButtonMenu.Fixed(openFolderBtn, 80)

	bx := fltk.NewBox(fltk.BORDER_BOX, 0, 0, 1, 1, "box")
	a.ButtonMenu.Add(bx)
	a.ButtonMenu.End()

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

// openFile prompts the user to select one or more video files, processes them,
// and adds them to the scrollable list.
func openFile() {
	// Create a new file chooser dialog for video files
	chooser := fltk.NewFileChooser(
		workingDir,                         // Default directory
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

	// Set up a new workflow to filter selected files
	wf := filter.NewWorkflow()
	wf.Add(filter.NewVideoFileFilter())

	list, err := wf.Process(chooser.Selection()) // Process selected files
	if err != nil {
		slog.Error("open files", "message", err)
		return
	}

	// If no valid video files found, log and return
	if len(list) == 0 {
		slog.Info("open files", "no video files selected")
		return
	}

	// Update working directory to the location of the first file
	workingDir, _ = filepath.Split(list[0])
	// Add each processed video file to the scrollable list
	for _, item := range list {
		lister.AddRow(util.GetInfoFromFileName(item))
	}
}

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
