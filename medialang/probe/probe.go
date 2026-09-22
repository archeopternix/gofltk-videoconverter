package probe

import (
	"context"
	"log/slog"

	"github.com/archeopternix/gofltk-videoconverter/medialang/media"
)

type Probe interface {
	Probe(ctx context.Context, filename string, logger *slog.Logger) (media.MediaSpec, error)
}
