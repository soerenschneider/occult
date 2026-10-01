package vault

// WithRevokeToken revokes the token after each request. Only use it with auth methods that issue a new token on
// login, such as AppRole.
func WithRevokeToken() VaultOpt {
	return func(v *Client) error {
		v.revokeToken = true
		return nil
	}
}
