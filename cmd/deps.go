package main

import (
	"log/slog"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/api/auth/approle"
	"github.com/soerenschneider/occult/v2/internal"
	"github.com/soerenschneider/occult/v2/internal/config"
	"github.com/soerenschneider/occult/v2/internal/vault"
	auth "github.com/soerenschneider/occult/v2/internal/vault/auth"
)

type dependencies struct {
	vaultAuth api.AuthMethod
	vault     internal.Vault

	occult *internal.Occult
}

func buildDeps(conf config.OccultConfig) *dependencies {
	deps := &dependencies{}
	err := deps.buildVaultAuth(conf)
	dieOnError(err, "can not vault auth")

	err = deps.buildVaultClient(conf)
	dieOnError(err, "can not vault client")

	deps.occult, err = internal.NewOccult(deps.vault, conf)
	dieOnError(err, "could not build occult")

	return deps
}

func dieOnError(err error, msg string) {
	if err != nil {
		slog.Error(msg, "error", err)
		os.Exit(1)
	}
}

func (d *dependencies) buildVaultAuth(conf config.OccultConfig) error {
	var err error
	switch conf.VaultAuth.Type {
	case config.VaultAuthApprole:
		secretId := &approle.SecretID{
			FromFile:   expandPath(conf.VaultAuth.ApproleSecretFile),
			FromString: conf.VaultAuth.ApproleSecret,
		}
		opts := []approle.LoginOption{
			approle.WithMountPath(conf.VaultAuth.ApproleMount),
		}
		d.vaultAuth, err = approle.NewAppRoleAuth(conf.VaultAuth.ApproleRoleId, secretId, opts...)
	default:
		d.vaultAuth = auth.NewTokenImplicitAuth()
	}
	return err
}

func (d *dependencies) buildVaultClient(conf config.OccultConfig) error {
	client, err := getVaultApiClient(conf.VaultAuth)
	if err != nil {
		return err
	}

	var opts []vault.VaultOpt
	if conf.VaultAuth.Type == config.VaultAuthApprole {
		// approle issues a new token on every login, the implicit token must not be revoked
		opts = append(opts, vault.WithRevokeToken())
	}

	d.vault, err = vault.New(client, d.vaultAuth, opts...)
	return err
}

func getVaultApiClient(conf config.VaultConfig) (*api.Client, error) {
	vaultConfig := api.DefaultConfig()
	vaultConfig.MaxRetries = 5
	vaultConfig.Address = conf.Address

	client, err := api.NewClient(vaultConfig)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func expandPath(path string) string {
	if !strings.Contains(path, "~") {
		return path
	}

	usr, err := user.Current()
	if err != nil {
		return path
	}

	dir := usr.HomeDir

	if path == "~" {
		return dir
	} else if strings.HasPrefix(path, "~/") {
		return filepath.Join(dir, path[2:])
	}

	return path
}
