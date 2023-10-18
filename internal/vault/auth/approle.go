package vault

import (
	"fmt"
	"os"

	"context"

	"github.com/hashicorp/vault/api"
	"github.com/soerenschneider/occult/v2/internal"
	"github.com/soerenschneider/occult/v2/internal/config"
)

const (
	KeyRoleId   = "role_id"
	KeySecretId = "secret_id"
)

type AppRoleAuth struct {
	conf config.VaultConfig
}

func NewAppRoleAuth(conf config.VaultConfig) (*AppRoleAuth, error) {
	return &AppRoleAuth{
		conf: conf,
	}, nil
}

func (t *AppRoleAuth) Cleanup(_ context.Context, client *api.Client) error {
	path := "auth/token/revoke-self"
	_, err := client.Logical().Write(path, map[string]any{})
	return err
}

func (t *AppRoleAuth) getSecretId() (string, error) {
	if len(t.conf.ApproleSecret) > 0 {
		return t.conf.ApproleSecret, nil
	}

	data, err := os.ReadFile(internal.ExpandTilde(t.conf.ApproleSecretFile))
	if err != nil {
		return "", fmt.Errorf("could not read secret_id from file %v", err)
	}

	return string(data), nil
}

func (t *AppRoleAuth) Login(ctx context.Context, client *api.Client) (*api.Secret, error) {
	secretId, err := t.getSecretId()
	if err != nil {
		return nil, fmt.Errorf("could not get login data: %v", err)
	}

	path := fmt.Sprintf("auth/%s/login", t.conf.ApproleMount)
	data := map[string]interface{}{
		KeyRoleId:   t.conf.ApproleRoleId,
		KeySecretId: secretId,
	}

	secret, err := client.Logical().Write(path, data)
	if err != nil {
		return nil, err
	}

	return secret, nil
}
