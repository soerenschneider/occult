package vault

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/hashicorp/vault/api"
)

const (
	tokenEnvVar = "VAULT_TOKEN"
	tokenFile   = ".vault-token"
)

type TokenImplicitAuth struct {
}

func NewTokenImplicitAuth() *TokenImplicitAuth {
	return &TokenImplicitAuth{}
}

func (t *TokenImplicitAuth) Login(_ context.Context, _ *api.Client) (*api.Secret, error) {
	token := os.Getenv(tokenEnvVar)
	if len(token) > 0 {
		slog.Info("Using vault token from env var", "env_var", tokenEnvVar)
		ret := &api.Secret{Auth: &api.SecretAuth{ClientToken: token}}
		return ret, nil
	}

	dirname, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("can't get user home dir: %v", err)
	}

	tokenPath := path.Join(dirname, tokenFile)
	if _, err := os.Stat(tokenPath); errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("file '%s' to read vault token from does not exist", tokenPath)
	}

	read, err := os.ReadFile(tokenPath)
	if err != nil {
		return nil, fmt.Errorf("error reading file '%s': %v", tokenPath, err)
	}

	slog.Info("Using vault token from file", "path", tokenPath)
	ret := &api.Secret{Auth: &api.SecretAuth{ClientToken: strings.TrimSpace(string(read))}}
	return ret, nil
}

func (t *TokenImplicitAuth) Cleanup(_ context.Context, _ *api.Client) error {
	return nil
}
