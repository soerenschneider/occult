package vault

func WithTransitPath(path string) VaultOpt {
	return func(v *Client) error {
		v.transitPath = path
		return nil
	}
}

func WithKv2Path(path string) VaultOpt {
	return func(v *Client) error {
		v.kv2Path = path
		return nil
	}
}

// WithRevokeToken revokes the token after each request. Only use it with auth methods that issue a new token on
// login, such as AppRole.
func WithRevokeToken() VaultOpt {
	return func(v *Client) error {
		v.revokeToken = true
		return nil
	}
}
