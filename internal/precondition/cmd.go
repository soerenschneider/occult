package precondition

import (
	"context"
	"os/exec"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/occult/v2/internal/config"
)

const defaultExitCode = 0

type CmdPrecondition struct {
	cmd           string
	wantsExitCode int
}

func NewCmd(config config.CmdPreconditionConfig) (*CmdPrecondition, error) {
	p := &CmdPrecondition{
		cmd:           config.Command,
		wantsExitCode: defaultExitCode,
	}

	if config.WantedExitCode != nil {
		p.wantsExitCode = *config.WantedExitCode
	}

	return p, nil
}

func (p *CmdPrecondition) ShouldPerformUnlock(ctx context.Context) bool {
	cmdWithArgs := strings.Split(p.cmd, " ")
	cmd := exec.CommandContext(ctx, cmdWithArgs[0], cmdWithArgs[1:]...) // #nosec: G204
	if err := cmd.Run(); err != nil {
		log.Debug().Err(err).Msgf("Running precondition yielded error")
	}

	return cmd.ProcessState.ExitCode() != p.wantsExitCode
}
