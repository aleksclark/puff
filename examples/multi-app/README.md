# Multi-App Example

This example demonstrates how to use Puff to manage configuration for multiple applications across different environments.

## Structure

This example has:
- 2 applications: `api` and `frontend`
- 2 environments: `dev` and `prod`
- Shared configuration in `base/`
- Environment-specific configuration in `dev/` and `prod/`

## Try it out

### View configurations

```bash
# View API configuration for dev
puff generate -a api -e dev -f env

# View API configuration for prod
puff generate -a api -e prod -f env

# View frontend configuration for dev
puff generate -a frontend -e dev -f env

# View frontend configuration for prod
puff generate -a frontend -e prod -f env
```

### Get specific values

```bash
# Get the PORT for api in dev
puff get -k PORT -a api -e dev

# Get the TITLE for frontend in dev (notice template resolution)
puff get -k TITLE -a frontend -e dev
```

### Generate different formats

```bash
# Generate JSON for API in production
puff generate -a api -e prod -f json

# Generate Kubernetes secret for API in production
puff generate -a api -e prod -f k8s --secret-name api-secret

# Generate .env file for frontend in dev
puff generate -a frontend -e dev -f env -o frontend-dev.env
```

## Key Features Demonstrated

1. **Shared Configuration**: `LOG_LEVEL` is defined once in `base/shared.yml` and inherited by all apps/envs

2. **Environment-Specific Values**: `DATABASE_URL` differs between `dev` and `prod`

3. **App-Specific Values**: `PORT` is only set for the `api` app, not `frontend`

4. **Template Variables**: `TITLE` in frontend uses `${_APP_TITLE}` and `${ENV}`

5. **Internal Variables**: `_APP_TITLE` (prefixed with `_`) is used in templates but not exported

## Configuration Files

```
.
├── base/
│   └── shared.yml          # Global config: LOG_LEVEL, _APP_TITLE
├── dev/
│   ├── shared.yml          # Dev env config: ENV, DEBUG, DATABASE_URL
│   ├── api.yml             # Dev API config: PORT=3000
│   └── frontend.yml        # Dev frontend config: TITLE with template
├── prod/
│   ├── shared.yml          # Prod env config: ENV, DEBUG, DATABASE_URL
│   ├── api.yml             # Prod API config: PORT=8080
│   └── frontend.yml        # Prod frontend config: TITLE with template
└── .sops.yaml              # SOPS configuration (for encryption)
```

## Expected Outputs

### API in Dev
```
DEBUG=true
DATABASE_URL="postgres://localhost/myapp_dev"
ENV=development
LOG_LEVEL=info
PORT=3000
```

### API in Prod
```
DATABASE_URL="postgres://prod-server/myapp"
DEBUG=false
ENV=production
LOG_LEVEL=info
PORT=8080
```

### Frontend in Dev
```
DATABASE_URL="postgres://localhost/myapp_dev"
DEBUG=true
ENV=development
LOG_LEVEL=info
TITLE="My Cool App (development)"
```

### Frontend in Prod
```
DATABASE_URL="postgres://prod-server/myapp"
DEBUG=false
ENV=production
LOG_LEVEL=info
TITLE="My Cool App"
```

Notice that `_APP_TITLE` doesn't appear in the output because it's an internal variable.
