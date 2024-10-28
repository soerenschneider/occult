package vault

import (
	"context"
	"errors"
	"net/url"

	"github.com/hashicorp/vault/api"
	"go.uber.org/multierr"
)

const (
	defaultTransitPath = "transit"
	defaultKv2Path     = "secret"
)

type Client struct {
	client *api.Client
	auth   api.AuthMethod

	transitPath string
	kv2Path     string
}

type VaultOpt func(v *Client) error

func New(client *api.Client, auth api.AuthMethod, opts ...VaultOpt) (*Client, error) {
	if client == nil {
		return nil, errors.New("empty client")
	}
	if auth == nil {
		return nil, errors.New("no auth")
	}

	c := &Client{
		client:      client,
		auth:        auth,
		kv2Path:     defaultKv2Path,
		transitPath: defaultTransitPath,
	}

	var errs error
	for _, opt := range opts {
		if err := opt(c); err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	return c, errs
}

var (
	ErrAuthFailed    = errors.New("auth failed")
	ErrNotFound      = errors.New("not found")
	ErrEmptySecrte   = errors.New("empty secret")
	ErrInvalidData   = errors.New("invalid data")
	ErrDecryptFailed = errors.New("decrypt failed")
)

func (v *Client) ReadKv2(ctx context.Context, path string) (map[string]any, error) {
	_, err := v.client.Auth().Login(ctx, v.auth)
	if err != nil {
		return nil, ErrAuthFailed
	}
	defer func() {
		_ = v.client.Auth().Token().RevokeSelf("xxx")
	}()

	secret, err := v.client.KVv2(v.kv2Path).Get(ctx, path)
	if err != nil {
		return nil, err
	}

	return secret.Data, nil
}

func (v *Client) ReadTransitSecret(ctx context.Context, path, ciphertext string) (map[string]any, error) {
	_, err := v.client.Auth().Login(ctx, v.auth)
	if err != nil {
		return nil, ErrAuthFailed
	}
	defer func() {
		_ = v.client.Auth().Token().RevokeSelf("xxx")
	}()

	decryptData := map[string]any{
		"ciphertext": ciphertext,
	}

	path, err = url.JoinPath(v.transitPath, "decrypt", path)
	if err != nil {
		return nil, err
	}

	response, err := v.client.Logical().WriteWithContext(ctx, path, decryptData)
	if err != nil {
		return nil, ErrDecryptFailed
	}

	return response.Data, nil
}
