package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

const (
	Kv2SecretType     = "kv2"
	TransitSecretType = "transit"

	VaultAuthImplicit = "implicit"
	VaultAuthApprole  = "approle"
	VaultAuthToken    = "token"

	DefaultApproleMount = "approle"
)

func Read(path string) (*OccultConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	conf := &OccultConfig{}
	return conf, yaml.Unmarshal(data, conf)
}

// OccultConfig holds the configuration to configure Occult.
type OccultConfig struct {
	UnlockRequests []UnlockConfig `yaml:"secrets" validate:"dive,required"`
	VaultAuth      VaultConfig    `yaml:"vault_auth" validate:"required"`

	// MetricsPath points to an optional Prometheus node_exporter textfile directory where metrics are stored.
	MetricsPath string `yaml:"metrics_path" validate:"omitempty,dirpath"`
}

// UnlockConfig describes how a thing is unlocked.
type UnlockConfig struct {
	// Profile denotes a nice name for this unlocker
	Profile string `yaml:"profile" validate:"required"`

	// SecretPath describes the relative path in Vault to retrieve the secret
	SecretPath string `yaml:"secret_path" validate:"required"`

	// SecretType can either be kv2 or transit and instructs whether to decrypt it using the transit secret engine or
	// read it from KV2.
	SecretType string `yaml:"secret_type" validate:"omitempty,oneof=kv2 transit"`

	// Accessor describes the key of the read secret to extract the secret value from.
	Accessor string `yaml:"accessor_path" validate:"required"`

	// CipherTextData contains the encrypted data that can be decrypted using the transit secret engine.
	CipherTextData string `yaml:"cipher_text" validate:"required_if=SecretType transit"`

	// Command instructs occult how to actually unlock something.
	Command string `yaml:"command" validate:"required"`

	// PostHooks can be set optionally and are run after unlocking.
	PostHooks []string `yaml:"post_hooks"`

	// PostHooksStopOnError describes whether to stop after a PostHook has encountered an error or whether to continue.
	PostHooksStopOnError bool `yaml:"post_hooks_stop_on_error"`

	// Precondition is an optional precondition that indicates whether unlocking is needed.
	Precondition *PreconditionConfigContainer `yaml:"precondition,omitempty"`
}

func (c *UnlockConfig) UnmarshalYAML(node *yaml.Node) error {
	type alias UnlockConfig

	tmp := &alias{
		SecretType:           Kv2SecretType,
		PostHooksStopOnError: true,
	}

	if err := node.Decode(&tmp); err != nil {
		return err
	}

	*c = UnlockConfig(*tmp)
	return nil
}

// VaultConfig is used to configure the connection against Vault.
type VaultConfig struct {
	// Address is the address of the Vault API endpoint.
	Address string `yaml:"address" validate:"required"`

	// Type describes what type of authentication against Vault is used.
	Type string `yaml:"auth_type" validate:"omitempty,oneof=token approle implicit"`

	// Token specifies the token use as authentication.
	Token string `yaml:"token" validate:"required_if=Type token TokenFile ''"`

	// TokenFile describes the file that contains the Vault token.
	TokenFile string `yaml:"token_file" validate:"required_if=Type token Token '',omitempty,file"`

	// ApproleRoleId describes the Vault Approle role_id used for login via Approle authentication.
	ApproleRoleId string `yaml:"approle_role_id" validate:"required_if=Type approle"`

	// ApproleSecret describes the Vault Approle secret_id used for login via Approle authentication.
	ApproleSecret string `yaml:"approle_secret_id" validate:"required_if=Type approle ApproleSecretFile ''"`

	// ApproleSecretFile describes the file that contains the Vault Approle secret_id used for login via Approle
	// authentication.
	ApproleSecretFile string `yaml:"approle_secret_id_file" validate:"required_if=Type approle ApproleSecret '',omitempty,filepath"`

	// ApproleMount is used to specify the path of the Approle auth.
	ApproleMount string `yaml:"approle_mount" validate:"excludes=/"`
}

func (c *VaultConfig) UnmarshalYAML(node *yaml.Node) error {
	type alias VaultConfig

	tmp := &alias{
		Type:         VaultAuthImplicit,
		ApproleMount: DefaultApproleMount,
	}

	if err := node.Decode(&tmp); err != nil {
		return err
	}

	*c = VaultConfig(*tmp)
	return nil
}
