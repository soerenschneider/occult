package internal

import (
	"os/user"
	"strings"
)

func ExpandTilde(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}

	usr, err := user.Current()
	if err != nil {
		return path
	}
	return usr.HomeDir + path[1:]
}
