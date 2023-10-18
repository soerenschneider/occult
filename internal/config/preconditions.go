package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

const (
	PreconditionPathType = "path"
	PreconditionCmdType  = "cmd"
)

type PreconditionConfig interface {
	GetType() string
}

type PreconditionConfigContainer struct {
	PreconditionConfig PreconditionConfig
}

func (c *PreconditionConfigContainer) UnmarshalYAML(node *yaml.Node) error {
	type inner struct {
		Type string `yaml:"type"`
	}

	hookType := &inner{}
	if err := node.Decode(hookType); err != nil {
		return err
	}

	switch hookType.Type {
	case PreconditionPathType:
		hook := &PathPreconditionConfig{}
		if err := node.Decode(hook); err != nil {
			return err
		}
		c.PreconditionConfig = hook

	case PreconditionCmdType:
		hook := &CmdPreconditionConfig{}
		if err := node.Decode(hook); err != nil {
			return err
		}
		c.PreconditionConfig = hook
	default:
		return fmt.Errorf("unknown hook type %q", hookType.Type)
	}

	if err := Validate(c.PreconditionConfig); err != nil {
		return err
	}

	return nil
}

func (w *CmdPreconditionConfig) GetType() string {
	return PreconditionCmdType
}

// CmdPreconditionConfig checks whether unlocking is necessary based on the exit code of a command.
type CmdPreconditionConfig struct {
	// Command specifies the command whose exit code is used to test whether to continue.
	Command string `yaml:"command" validate:"required"`
	// WantedExitCode gives the ability to specify a concrete exit code that we expect.
	WantedExitCode *int `yaml:"exit_code" validate:"omitempty,number"`
}

func (w *PathPreconditionConfig) GetType() string {
	return PreconditionPathType
}

// PathPreconditionConfig checks whether unlocking is necessary based on the absence or presence of a path.
type PathPreconditionConfig struct {
	// Path is the path under test
	Path string `yaml:"path" validate:"required,filepath"`
	//WantsAbsence if true and path exists -> unlocking continues.
	WantsAbsence *bool `yaml:"absent" validate:"omitempty"`
}
