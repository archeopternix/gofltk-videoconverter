package main

import (
	"log/slog"

	"github.com/archeopternix/gofltk-videoconverter/ui"
	"github.com/pwiecz/go-fltk"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	window := fltk.NewWindow(500, 440)
	app := ui.NewApp(window)

	window.Show()

	fltk.Run()
}
