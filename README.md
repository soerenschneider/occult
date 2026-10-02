# occult
[![Go Report Card](https://goreportcard.com/badge/github.com/soerenschneider/occult/v2)](https://goreportcard.com/report/github.com/soerenschneider/occult/v2)
![test-workflow](https://github.com/soerenschneider/occult/actions/workflows/test.yaml/badge.svg)
![release-workflow](https://github.com/soerenschneider/occult/actions/workflows/release.yaml/badge.svg)
![golangci-lint-workflow](https://github.com/soerenschneider/occult/actions/workflows/golangci-lint.yaml/badge.svg)

Generic process automation based on reading secrets from Vault and piping them to other tools

## Features

🔐 Reads secret data from Vault and pipes it to a pre-defined program<br/>
🛂 Authenticates via AppRole or an existing token<br/>
🔑 Reads secrets from KV2 or decrypts them using the transit secret engine<br/>
🚦 Supports preconditions to skip unlocking when not needed<br/>
🪝 Supports running post-hooks<br/>
🔭 Observability using Prometheus metrics<br/>

## Use-Cases

🔓 Unlock encrypted disks automatically right after system boot<br/>
🔓 Unlock password protected keys automatically<br/>

## Installation

```shell
$ git clone https://github.com/soerenschneider/occult.git
$ cd occult
$ make build
```

## Configuration

Occult reads a YAML config file. If no path is given via `-config`, the first existing file of the following locations is used:

1. `occult.yaml` (current working directory)
2. `~/.occult.yaml`
3. `/etc/occult.yaml`

### Command line flags

| Flag       | Description                                  |
|------------|----------------------------------------------|
| `-config`  | Path to the config file (`~` gets expanded)  |
| `-debug`   | Enable debug logging                         |
| `-version` | Print the version and exit                   |

### Example

```yaml
vault_auth:
  address: https://vault.example.com:8200
  auth_type: approle
  approle_role_id: 2f1c9e3a-...
  approle_secret_id_file: ~/.occult-secret-id

metrics_path: /var/lib/node_exporter

secrets:
  # Read a secret from KV2 and pipe it to cryptsetup, but only if the device is not unlocked yet
  - profile: data-disk
    secret_path: occult/data-disk
    accessor_path: passphrase
    command: cryptsetup open /dev/sdb1 data --key-file=-
    precondition:
      type: path
      path: /dev/mapper/data
      absent: true
    post_hooks:
      - mount /dev/mapper/data /mnt/data
      - systemctl restart some-service

  # Decrypt a ciphertext using the transit secret engine
  - profile: ssh-key
    secret_type: transit
    transit_key: occult
    cipher_text: "vault:v1:8SDd3WHDOjf7mq69CyCqYjBXAiQQAVZRkFM13ok481zoCmHnSeDX9vyf7w=="
    command: ssh-add -
    precondition:
      type: cmd
      command: ssh-add -l
```

### Reference

#### Top level

| Key            | Description                                                                                                                                 | Type                          | Mandatory | Default |
|----------------|---------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------|-----------|---------|
| `vault_auth`   | Connection and authentication settings for Vault                                                                                            | [vault_auth](#vault_auth)     | Y         |         |
| `secrets`      | List of secrets to unlock. They are processed sequentially, each with a timeout of one minute                                               | list of [secrets](#secrets)   | Y         |         |
| `metrics_path` | Existing directory to write `occult.prom` to. Point it to your [node_exporter textfile](https://github.com/prometheus/node_exporter#textfile-collector) directory | string           | N         |         |

#### `vault_auth`

| Key                      | Description                                                                 | Type   | Mandatory               | Default    |
|--------------------------|-----------------------------------------------------------------------------|--------|-------------------------|------------|
| `address`                | Address of the Vault API                                                    | string | Y                       |            |
| `auth_type`              | Authentication method, either `implicit` or `approle`                       | string | N                       | `implicit` |
| `approle_role_id`        | AppRole `role_id`                                                           | string | if `approle`            |            |
| `approle_secret_id`      | AppRole `secret_id`                                                         | string | if `approle`&ast;       |            |
| `approle_secret_id_file` | File to read the AppRole `secret_id` from (`~` gets expanded)               | string | if `approle`&ast;       |            |
| `approle_mount`          | Mount path of the AppRole auth method (must not contain `/`)                | string | N                       | `approle`  |

&ast; Provide *either* `approle_secret_id` or `approle_secret_id_file`.

**Authentication methods**

- `implicit`: Uses an existing token, read from the `VAULT_TOKEN` environment variable or, as fallback, from `~/.vault-token`. The token is not revoked after use.
- `approle`: Logs in via AppRole. The issued token is revoked after the secret has been read.

#### `secrets`

| Key                        | Description                                                                                       | Type                            | Mandatory          | Default    |
|----------------------------|---------------------------------------------------------------------------------------------------|---------------------------------|--------------------|------------|
| `profile`                  | Name of this secret, used in logs and metric labels                                               | string                          | Y                  |            |
| `command`                  | Command (and arguments) that receives the secret via stdin                                        | string                          | Y                  |            |
| `secret_type`              | Where to get the secret from, either `kv2` or `transit`                                           | string                          | N                  | `kv2`      |
| `kv2_mount`                | Mount path of the KV2 secret engine                                                               | string                          | N                  | `secret`   |
| `secret_path`              | Path of the secret, relative to `kv2_mount`                                                       | string                          | if `kv2`           |            |
| `accessor_path`            | Key within the secret's data that holds the value to pipe                                         | string                          | if `kv2`           |            |
| `transit_mount`            | Mount path of the transit secret engine                                                           | string                          | N                  | `transit`  |
| `transit_key`              | Name of the transit key used for decryption                                                       | string                          | if `transit`       |            |
| `cipher_text`              | Ciphertext to decrypt, must start with `vault:v` (e.g. `vault:v1:...`)                            | string                          | if `transit`       |            |
| `precondition`             | Check that decides whether unlocking is necessary at all                                          | [precondition](#precondition)   | N                  |            |
| `post_hooks`               | Commands to run after the secret has been piped successfully                                      | list of strings                 | N                  |            |
| `post_hooks_stop_on_error` | Whether to skip the remaining post hooks after one has failed                                     | bool                            | N                  | `true`     |

Commands (`command`, `post_hooks`, precondition `command`) are split on spaces and executed directly, *not* via a shell. Quoting, pipes and redirects are therefore not supported; wrap them in a script if needed.

#### `precondition`

A precondition is evaluated before contacting Vault. If it indicates that nothing needs to be unlocked, the secret is skipped.

| Key         | Description                                                                                                  | Type   | Mandatory    | Default |
|-------------|--------------------------------------------------------------------------------------------------------------|--------|--------------|---------|
| `type`      | Either `path` or `cmd`                                                                                       | string | Y            |         |
| `path`      | `path`: path to check for                                                                                    | string | if `path`    |         |
| `absent`    | `path`: if `false`, unlock when the path exists. If `true`, unlock when the path does *not* exist            | bool   | N            | `false` |
| `command`   | `cmd`: command to run                                                                                        | string | if `cmd`     |         |
| `exit_code` | `cmd`: exit code signaling that unlocking is *not* necessary. Any other exit code triggers unlocking         | int    | N            | `0`     |

## Logging

Occult logs to stderr. When stdout is a terminal, logs are printed as human-readable text, otherwise as JSON (one object per line), which is suited for journald or log shippers. Use `-debug` to enable debug logs.

## Metrics

Metrics are only written if `metrics_path` is set. The file is written atomically, so the node_exporter textfile collector never reads a partially written file.

| Name                           | Help                                                                  | Type  | Labels        |
|--------------------------------|-----------------------------------------------------------------------|-------|---------------|
| occult_last_invocation_seconds | Unix timestamp of the last run of a profile                           | Gauge | profile       |
| occult_success_bool            | Whether the profile ran successfully, including its post hooks        | Gauge | profile       |
| occult_unlocked_bool           | Whether the unlock command ran successfully (`0` if it was skipped)   | Gauge | profile       |
| occult_post_hook_success       | Whether a post hook ran successfully                                  | Gauge | profile, hook |

A profile that is skipped because of its precondition counts as successful, but reports `occult_unlocked_bool 0`. The `hook` label contains the full post hook command.

## Vault Example Configuration

Terraform snippet to configure Vault accordingly

```hcl
resource "vault_policy" "occult" {
  name = "occult_${var.secret_name}"

  policy = <<EOT
path "secret/data/occult/${var.secret_name}" {
  capabilities = ["read"]
}
EOT
}

resource "vault_token_auth_backend_role" "occult" {
  role_name = vault_policy.occult.name
  allowed_policies = [
    vault_policy.occult.name,
    "default"
  ]
  orphan = true
}

resource "vault_token" "occult" {
  depends_on = [vault_token_auth_backend_role.occult]
  role_name  = vault_policy.occult.name
  policies = [
    vault_policy.occult.name
  ]
  display_name = "occult-${var.secret_name}"
  renewable    = true
  ttl          = var.token_ttl
}
```
