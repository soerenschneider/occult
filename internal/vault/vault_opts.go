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
