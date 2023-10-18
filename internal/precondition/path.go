package precondition

import (
	"context"
	"errors"
	"os"

	"github.com/soerenschneider/occult/v2/internal/config"
)

type PathPrecondition struct {
	path         string
	wantsAbsence bool
}

func NewPathPrecondition(config config.PathPreconditionConfig) (*PathPrecondition, error) {
	p := &PathPrecondition{
		path:         config.Path,
		wantsAbsence: false,
	}

	if config.WantsAbsence != nil {
		p.wantsAbsence = *config.WantsAbsence
	}

	return p, nil
}

func (p *PathPrecondition) ShouldPerformUnlock(ctx context.Context) bool {
	_, err := os.Stat(p.path)
	if errors.Is(err, os.ErrNotExist) {
		return p.wantsAbsence
	}

	return !p.wantsAbsence
}
