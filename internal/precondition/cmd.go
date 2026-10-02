package precondition

import (
	"context"
	"log/slog"
	"os/exec"
	"strings"

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
		slog.Debug("Running precondition yielded error", "error", err)
	}

	return cmd.ProcessState.ExitCode() != p.wantsExitCode
}
