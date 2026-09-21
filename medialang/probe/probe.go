package probe

import (
	"context"

	"github.com/archeopternix/gofltk-videoconverter/medialang/media"
)

type Probe interface {
	Probe(ctx context.Context, filename string) (media.MediaSpec, error)
}
