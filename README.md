# Puff

A dead-simple GitOps secret and environment variable management tool.

Puff provides a DRY (Don't Repeat Yourself) way to manage configuration across multiple applications, environments, and deployment targets while keeping secrets secure with SOPS encryption.

## Features

- **Multi-dimensional configuration**: Organize configs by application, environment, and target
- **6-level precedence system**: Base → App → Env → Env+App → Target → Target+App
- **Template variables**: Reference other variables with `${VAR}` syntax
- **Internal variables**: Use `_` prefix for variables that shouldn't be exported
- **Multiple output formats**: .env, JSON, YAML, and Kubernetes secrets
- **SOPS integration**: Secure encryption with age keys - fully integrated
- **Single binary**: No dependencies to install

## Installation

### From Source

```bash
go install github.com/teamcurri/puff@latest
```

### Manual Build

```bash
git clone https://github.com/teamcurri/puff
cd puff
go build -o puff .
```

## Quick Start

### 1. Initialize a new configuration directory

First, generate an age key pair for encryption:
```bash
# Install age if you haven't already
# Ubuntu/Debian: sudo apt-get install age
# macOS: brew install age

# Generate a key pair
age-keygen -o key.txt
```

Then initialize puff with your age public key:
```bash
mkdir my-config && cd my-config
puff init --age-keys "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p"
```

This creates:
- `base/shared.yml` - Global configuration (encrypted)
- `.sops.yaml` - SOPS configuration with your age key

**Important**: Save your private key (`key.txt`) securely. Set the environment variable to decrypt files:
```bash
export SOPS_AGE_KEY_FILE=~/key.txt
```

### 2. Add configuration values

```bash
# Add a global variable
puff set -k LOG_LEVEL -v info

# Add an app-specific variable
puff set -k APP_NAME -v api -a api

# Add an environment-specific variable
puff set -k DATABASE_URL -v "postgres://localhost/dev" -e dev

# Add an environment + app-specific variable
puff set -k PORT -v 8080 -a api -e dev
```

### 3. Generate configuration

```bash
# Generate .env file
puff generate -a api -e dev -f env

# Generate JSON
puff generate -a api -e dev -f json

# Generate Kubernetes secret
puff generate -a api -e dev -f k8s --secret-name api-secret
```

## Configuration Structure

Puff organizes configuration files in a hierarchical structure:

```
my-config/
├── base/
│   ├── shared.yml          # Global config for all apps/envs
│   ├── api.yml             # Base config for 'api' app
│   └── worker.yml          # Base config for 'worker' app
├── dev/
│   ├── shared.yml          # Shared config for dev environment
│   ├── api.yml             # Dev-specific config for api
│   └── worker.yml          # Dev-specific config for worker
├── prod/
│   ├── shared.yml          # Shared config for prod environment
│   ├── api.yml             # Prod-specific config for api
│   └── worker.yml          # Prod-specific config for worker
├── target-overrides/
│   ├── docker/
│   │   ├── shared.yml      # Docker-specific overrides (all apps)
│   │   └── api.yml         # Docker-specific overrides for api
│   └── kubernetes/
│       ├── shared.yml      # K8s-specific overrides (all apps)
│       └── api.yml         # K8s-specific overrides for api
└── .sops.yaml              # SOPS encryption configuration
```

## Precedence Order

Configuration is merged with the following precedence (lowest to highest):

1. `base/shared.yml` - Truly global configuration
2. `base/{app}.yml` - Base app-specific configuration
3. `{env}/shared.yml` - Environment-wide configuration
4. `{env}/{app}.yml` - Environment + app-specific configuration
5. `target-overrides/{target}/shared.yml` - Target-wide overrides
6. `target-overrides/{target}/{app}.yml` - Target + app-specific overrides

Later values override earlier ones.

## Template Variables

Puff supports variable substitution using `${VAR}` syntax:

```yaml
# base/shared.yml
APP_TITLE: My Cool App
API_URL: https://api.example.com

# dev/frontend.yml
TITLE_TEXT: ${APP_TITLE} (Development)
API_ENDPOINT: ${API_URL}/v1
```

When generating config for `frontend` in `dev`:
```
APP_TITLE="My Cool App"
API_URL="https://api.example.com"
TITLE_TEXT="My Cool App (Development)"
API_ENDPOINT="https://api.example.com/v1"
```

### Internal Variables

Variables prefixed with `_` are available for templating but not exported:

```yaml
# base/shared.yml
_BASE_URL: https://api.example.com

# dev/api.yml
PUBLIC_URL: ${_BASE_URL}/public
ADMIN_URL: ${_BASE_URL}/admin
```

Output only includes `PUBLIC_URL` and `ADMIN_URL`, not `_BASE_URL`.

## Commands

### `init`

Initialize a new puff configuration directory with encryption.

```bash
puff init --age-keys "age1..." [OPTIONS]
```

Options:
- `-k, --age-keys`: Age public keys for encryption (required, comma-separated)
- `-d, --dir`: Directory to initialize (default: current directory)

Example:
```bash
# Single key
puff init --age-keys "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p"

# Multiple keys
puff init --age-keys "age1...,age1..."
```

### `set`

Set a configuration value.

```bash
puff set -k KEY -v VALUE [OPTIONS]
```

Options:
- `-k, --key`: Key to set (required)
- `-v, --value`: Value to set (required)
- `-a, --app`: Application name
- `-e, --env`: Environment name
- `-t, --target`: Target platform
- `-r, --root`: Root directory for config files (default: current directory)

The file location is determined by the flags:
- No flags: `base/shared.yml`
- `--app` only: `base/{app}.yml`
- `--env` only: `{env}/shared.yml`
- `--app --env`: `{env}/{app}.yml`
- `--target`: `target-overrides/{target}/shared.yml` or `{target}/{app}.yml`

### `get`

Get a configuration value.

```bash
puff get -k KEY [OPTIONS]
```

Options:
- `-k, --key`: Key to retrieve (required)
- `-a, --app`: Application name
- `-e, --env`: Environment name
- `-t, --target`: Target platform
- `-r, --root`: Root directory for config files (default: current directory)

Returns the resolved value after applying all merges and templates.

### `generate`

Generate full configuration in the specified format.

```bash
puff generate -a APP -e ENV -f FORMAT [OPTIONS]
```

Options:
- `-a, --app`: Application name (required)
- `-e, --env`: Environment name (required)
- `-f, --format`: Output format: `env`, `json`, `yaml`, `k8s` (required)
- `-t, --target`: Target platform (default: "local")
- `-o, --output`: Output file (default: stdout)
- `--secret-name`: Kubernetes secret name (required for k8s format)
- `--base64`: Base64 encode values for k8s secrets
- `-r, --root`: Root directory for config files (default: current directory)

Examples:
```bash
# Generate .env file
puff generate -a api -e prod -f env -o .env

# Generate JSON to stdout
puff generate -a api -e dev -f json

# Generate Kubernetes secret
puff generate -a api -e prod -f k8s --secret-name api-secret -o secret.yaml

# Generate with base64 encoding
puff generate -a api -e prod -f k8s --secret-name api-secret --base64
```

### `keys`

Manage encryption keys (SOPS integration).

#### `keys list`

List all encryption keys used in the configuration.

```bash
puff keys list [--root DIR]
```

Shows all age keys, the environments they're used in, and any associated comments.

#### `keys add`

Add an age encryption key to all files (or specific environment).

```bash
puff keys add -k KEY [OPTIONS]
```

Options:
- `-k, --key`: Age public key to add (required)
- `-c, --comment`: Comment for the key (e.g., "Bob's laptop")
- `-e, --env`: Only add to specific environment
- `-r, --root`: Root directory for config files (default: current directory)

Examples:
```bash
# Add key to all files
puff keys add -k "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p" -c "Alice's laptop"

# Add key only to prod environment
puff keys add -k "age1..." -e prod -c "Production team key"
```

The key is added to `.sops.yaml` and all encrypted files are re-encrypted with the new key included.

#### `keys rm`

Remove an age encryption key from all files (or specific environment).

```bash
puff keys rm -k KEY [OPTIONS]
```

Options:
- `-k, --key`: Age public key to remove (required)
- `-e, --env`: Only remove from specific environment
- `-r, --root`: Root directory for config files (default: current directory)

Examples:
```bash
# Remove key from all files
puff keys rm -k "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p"

# Remove key only from dev environment
puff keys rm -k "age1..." -e dev
```

**Note**: You cannot remove the last key from a file. At least one key must remain for encryption.

### `decrypt`

Decrypt a file for bulk editing.

```bash
puff decrypt -f FILE
```

Options:
- `-f, --file`: File to decrypt (required)

Creates a `.dec` version of the file that you can edit in plain text:
```bash
puff decrypt -f base/shared.yml
# Creates base/shared.dec.yml
vim base/shared.dec.yml
puff encrypt -f base/shared.dec.yml
# Re-encrypts and removes .dec file
```

### `encrypt`

Re-encrypt a decrypted file.

```bash
puff encrypt -f FILE
```

Options:
- `-f, --file`: Decrypted file to encrypt (must have .dec extension)

Re-encrypts the file using the keys from the original file (or `.sops.yaml`), then removes the `.dec` file for security.

## Output Formats

### .env Format

Standard environment variable format. Nested values are JSON-encoded.

```bash
puff generate -a api -e dev -f env
```

Output:
```
DATABASE_URL="postgres://localhost/dev"
PORT=8080
```

### JSON Format

```bash
puff generate -a api -e dev -f json
```

Output:
```json
{
  "DATABASE_URL": "postgres://localhost/dev",
  "PORT": 8080
}
```

### YAML Format

```bash
puff generate -a api -e dev -f yaml
```

Output:
```yaml
DATABASE_URL: postgres://localhost/dev
PORT: 8080
```

### Kubernetes Secret Format

```bash
puff generate -a api -e prod -f k8s --secret-name api-secret
```

Output:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: api-secret
stringData:
  DATABASE_URL: postgres://localhost/prod
  PORT: "8080"
```

With base64 encoding:
```bash
puff generate -a api -e prod -f k8s --secret-name api-secret --base64
```

Output:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: api-secret
type: Opaque
data:
  DATABASE_URL: cG9zdGdyZXM6Ly9sb2NhbGhvc3QvcHJvZA==
  PORT: ODA4MA==
```

## Best Practices

### 1. Use Internal Variables for DRY Configuration

```yaml
# base/shared.yml
_DOMAIN: example.com
_API_VERSION: v2

# prod/api.yml
API_URL: https://api.${_DOMAIN}/${_API_VERSION}
WEBHOOK_URL: https://webhooks.${_DOMAIN}/${_API_VERSION}
```

### 2. Organize by Environment First

Create separate directories for each environment (dev, staging, prod) rather than putting everything in base/.

### 3. Use Target Overrides Sparingly

Target overrides should only contain values that truly differ between deployment targets (e.g., Docker vs. Kubernetes). Most configuration should be in environment-specific files.

### 4. Keep Secrets Encrypted

All configuration files are automatically encrypted by puff using SOPS and age. Never commit unencrypted secrets or `.dec` files to git.

**Key Management Tips:**
- Store private keys securely (password manager, encrypted disk)
- Use separate keys for different environments when possible
- Rotate keys periodically using `puff keys add` and `puff keys rm`
- Add team members' keys with `puff keys add -c "Team member name"`

### 5. Document Your Variables

Add comments to configuration files explaining what each variable does:

```yaml
# Database configuration
DATABASE_URL: postgres://localhost/dev  # Local dev database

# API settings
API_TIMEOUT: 30  # Timeout in seconds for API calls
```

## Examples

### Multi-Environment Application

```bash
# Initialize
puff init

# Set global defaults
puff set -k LOG_LEVEL -v info
puff set -k _APP_NAME -v "My App"

# Development environment
puff set -k ENV_NAME -v development -e dev
puff set -k DEBUG -v true -e dev
puff set -k DATABASE_URL -v "postgres://localhost/myapp_dev" -e dev

# Production environment
puff set -k ENV_NAME -v production -e prod
puff set -k DEBUG -v false -e prod
puff set -k DATABASE_URL -v "postgres://prod-db/myapp" -e prod

# App-specific settings
puff set -k PORT -v 3000 -a api -e dev
puff set -k PORT -v 8080 -a api -e prod

# Generate configs
puff generate -a api -e dev -f env -o dev.env
puff generate -a api -e prod -f k8s --secret-name api-prod -o k8s-secret.yaml
```

### Using Templates

```bash
# Set base configuration with templates
puff set -k _CLUSTER_REGION -v us-west-2
puff set -k _SERVICE_NAME -v myapp

# Use templates in environment configs
puff set -k S3_BUCKET -v '${_SERVICE_NAME}-${_CLUSTER_REGION}-assets' -e prod
puff set -k CLOUDFRONT_URL -v 'https://${_SERVICE_NAME}.cloudfront.net' -e prod

# Generate - internal variables won't be exported
puff generate -a api -e prod -f env
# Output:
# S3_BUCKET=myapp-us-west-2-assets
# CLOUDFRONT_URL=https://myapp.cloudfront.net
```

## Testing

Run unit tests:
```bash
go test ./internal/...
```

Run integration tests:
```bash
go test ./test/...
```

Run all tests:
```bash
go test ./...
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see LICENSE file for details.
