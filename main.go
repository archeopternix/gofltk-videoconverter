package main

import (
	"log/slog"

	"github.com/archeopternix/gofltk-videoconverter/ui"
	"github.com/pwiecz/go-fltk"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	window := fltk.NewWindow(500, 440, "Video Enhancer and Converter")
	window.Resizable(window)
	app := ui.NewApp(window)
	app.Hello()
	window.Show()

	fltk.Run()
}
