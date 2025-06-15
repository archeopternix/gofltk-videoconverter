package ui

import (
	"fmt"
	"log/slog"
	"sort"

	"github.com/archeopternix/gofltk-videoconverter/util"
	"github.com/pwiecz/go-fltk"
)

//	"github.com/archeopternix/fltk/util"

// Define the size of the scrollbar
const (
	scrollbarSize = 16
	gap           = 10
)

// Row represents a single row in the scrollable area.
type Row struct {
	group     *fltk.Group       // Group container for row components
	checkbox  *fltk.CheckButton // Checkbox for row selection
	image     *fltk.Box         // Box for displaying an image or color
	namelabel *fltk.Box         // Label for displaying the name
	label     *fltk.Box         // Label for additional details
	button    *fltk.Button      // Button for performing an action
	info      *util.Info        // Associated info object
}

// NewRow creates and returns a new Row instance with the specified info.
func NewRow(info *util.Info) *Row {
	rowHeight := 50
	row := fltk.NewGroup(0, 0, 330, rowHeight)
	row.Begin()

	// Create and configure row widgets
	cb := fltk.NewCheckButton(10, 10, 20, 20, "")
	img := fltk.NewBox(fltk.FLAT_BOX, 40, 5, 40, 40, "")
	img.SetColor(fltk.BLUE)
	namelbl := fltk.NewBox(fltk.NO_BOX, 90, 5, 150, 20, fmt.Sprintf("%s", info.Name))
	namelbl.SetAlign(fltk.ALIGN_LEFT | fltk.ALIGN_INSIDE) // Align text left and inside the box

	lbl := fltk.NewBox(fltk.NO_BOX, 90, 22, 150, 37, fmt.Sprintf("(%dx%d / %s FPS)", info.ResolutionX, info.ResolutionY, info.FPS))
	lbl.SetAlign(fltk.ALIGN_LEFT | fltk.ALIGN_INSIDE) // Align text left and inside the box

	btn := fltk.NewButton(250, 10, 70, 30, "Info...")
	btn.SetCallback(func() {
		fmt.Println("Video Info Dialog called")
		//		videoInfoDialog(info)
	})

	row.End()

	// Return the new Row instance
	return &Row{
		group:     row,
		checkbox:  cb,
		image:     img,
		namelabel: namelbl,
		label:     lbl,
		button:    btn,
		info:      info,
	}
}

// Refresh updates the position and size of the row and its components.
func (r *Row) Refresh(x, y, width int) {
	rowHeight := 48
	r.group.Resize(x, y, width, rowHeight)
	r.checkbox.Resize(gap, y+10, 20, 20)
	r.image.Resize(30+gap, y+5, 40, 40)
	r.namelabel.Resize(70+gap, y+5, width-70-10-15-90, 20)
	r.label.Resize(70+gap, y+22, width-70-2*gap-90, 20)
	r.button.Resize(width-70-2*gap, y+10, 70, 30)

	// Make components visible
	r.checkbox.Show()
	r.image.Show()
	r.namelabel.Show()
	r.label.Show()
	r.button.Show()
	r.group.Show()
}

// Scroll represents a scrollable container with rows.
type Scroll struct {
	fltkScroll *fltk.Scroll // The underlying FLTK scroll widget
	rows       []*Row       // List of rows in the scroll
	lastW      int          // Last recorded width of the scroll container
	lastH      int          // Last recorded height of the scroll container
}

// NewScroll creates a new Scroll instance with specified dimensions.
func NewScroll(x, y, w, h int) *Scroll {
	scroll := fltk.NewScroll(x, y, w, h)
	scroll.Begin()
	scroll.End()

	s := &Scroll{
		fltkScroll: scroll,
		rows:       []*Row{},
		lastW:      w,
		lastH:      h,
	}

	// Start monitoring for resize events
	s.startResizeWatcher()

	return s
}

// AddRow adds a new Row to the scroll container.
func (s *Scroll) AddRow(info *util.Info) {
	// Avoid adding duplicate rows
	for _, r := range s.rows {
		if r.info.FullPath == info.FullPath {
			slog.Debug("duplicate row", "filepath", info.FullPath)
			return
		}
	}

	// Create a new row
	row := NewRow(info)

	// Add the row to the scroll container
	s.fltkScroll.Begin()
	s.fltkScroll.Add(row.group)
	s.rows = append(s.rows, row)
	s.Refresh()
	s.fltkScroll.End()
	s.fltkScroll.Redraw()
	slog.Debug("row added", "filepath", info.FullPath)
}

// DeleteRow removes a row at the specified index.
func (s *Scroll) DeleteRow(index int) {
	if index < 0 || index >= len(s.rows) {
		return // Invalid index
	}

	// Hide all components of the row
	row := s.rows[index]
	row.group.Hide()
	row.checkbox.Hide()
	row.image.Hide()
	row.namelabel.Hide()
	row.label.Hide()
	row.button.Hide()

	slog.Debug("row deleted", "filepath", row.info.FullPath)

	// Remove the row from the list
	s.rows = append(s.rows[:index], s.rows[index+1:]...)

	// Refresh the scroll container after deletion
	s.fltkScroll.Begin()
	s.Refresh()
	s.fltkScroll.End()
	s.fltkScroll.Redraw()

}

// GetSelectedRows returns the indices of selected rows in reverse order
// to support deletion of multiple files.
func (s *Scroll) GetSelectedRows() []int {
	var selected []int
	for i, r := range s.rows {
		if r.checkbox.Value() {
			selected = append(selected, i)
		}
	}
	// Sort the indices in descending order
	sort.Sort(sort.Reverse(sort.IntSlice(selected)))

	return selected
}

// startResizeWatcher monitors the scroll container for size changes and triggers a refresh.
func (s *Scroll) startResizeWatcher() {
	const interval = 0.1 // Check every 200ms

	// Add a timeout to monitor for changes in width and height
	fltk.AddTimeout(interval, func() {
		curW := s.fltkScroll.W()
		curH := s.fltkScroll.H()

		// Trigger a refresh if dimensions have changed
		if curW != s.lastW || curH != s.lastH {
			s.lastW = curW
			s.lastH = curH
			s.fltkScroll.Begin()
			s.Refresh()
			s.fltkScroll.End()
			s.fltkScroll.Redraw()
		}

		// Restart the watcher
		s.startResizeWatcher()
	})
}

// Refresh rearranges the rows in the scroll container and adjusts dimensions.
func (s *Scroll) Refresh() {
	y := 10 + 25 // Starting Y-coordinate for the first row

	// Estimate the total height of the content
	contentHeight := len(s.rows)*50 + 10

	// Adjust for the vertical scrollbar if needed
	scrollbarWidth := 0
	if contentHeight > s.fltkScroll.H() {
		scrollbarWidth = scrollbarSize
	}

	width := s.fltkScroll.W() - scrollbarWidth

	// Resize and show each row independently
	for _, r := range s.rows {
		r.Refresh(0, y, width)
		y += 50
	}
}

// GetFiles gets all selected files
func (s Scroll) GetSelectedFilePaths() []string {
	var files []string

	rows := s.rows
	for i := 0; i < len(rows); i++ {
		if rows[i].checkbox.IsActive() {
			info := rows[i].info
			files = append(files, info.FullPath) // Assumes info has a FilePath field
		}
	}
	return files
}
