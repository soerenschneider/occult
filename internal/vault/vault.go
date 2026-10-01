package vault

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"

	"github.com/hashicorp/vault/api"
	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
)

type Client struct {
	client *api.Client
	auth   api.AuthMethod

	// revokeToken controls whether the token is revoked after each request. Only enable it for auth methods that
	// issue a new token on login, otherwise the user's own token gets revoked.
	revokeToken bool
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
		client: client,
		auth:   auth,
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

func (v *Client) revokeTokenIfNeeded(ctx context.Context) {
	if !v.revokeToken {
		return
	}
	if err := v.client.Auth().Token().RevokeSelfWithContext(ctx, ""); err != nil {
		log.Warn().Err(err).Msg("could not revoke token")
	}
}

// ReadKv2 reads the secret at the given path from the KV2 secret engine mounted at mount.
func (v *Client) ReadKv2(ctx context.Context, mount, path string) (map[string]any, error) {
	_, err := v.client.Auth().Login(ctx, v.auth)
	if err != nil {
		return nil, ErrAuthFailed
	}
	defer v.revokeTokenIfNeeded(ctx)

	secret, err := v.client.KVv2(mount).Get(ctx, path)
	if err != nil {
		return nil, err
	}

	return secret.Data, nil
}

// ReadTransitSecret decrypts the ciphertext using the transit key with the given name from the transit secret engine
// mounted at mount.
func (v *Client) ReadTransitSecret(ctx context.Context, mount, key, ciphertext string) (string, error) {
	if key == "" {
		return "", errors.New("empty transit key")
	}

	_, err := v.client.Auth().Login(ctx, v.auth)
	if err != nil {
		return "", ErrAuthFailed
	}
	defer v.revokeTokenIfNeeded(ctx)

	decryptData := map[string]any{
		"ciphertext": ciphertext,
	}

	path, err := url.JoinPath(mount, "decrypt", key)
	if err != nil {
		return "", err
	}

	response, err := v.client.Logical().WriteWithContext(ctx, path, decryptData)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrDecryptFailed, err)
	}
	if response == nil || response.Data == nil {
		return "", ErrEmptySecrte
	}

	// transit returns the plaintext base64-encoded
	encoded, ok := response.Data["plaintext"].(string)
	if !ok {
		return "", fmt.Errorf("%w: no plaintext in response", ErrInvalidData)
	}
	plaintext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("%w: could not decode plaintext: %w", ErrInvalidData, err)
	}

	return string(plaintext), nil
}
