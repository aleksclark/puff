Project name: Puff

This project aims to provide a dead-simple, gitops secret+env var management tool.

It will be written in go, and aim to produce a single installable binary with no other dependencies.

Inspired by sops (https://github.com/getsops/sops), will use sops as a library to manage storing its data.

This tool will allow users to create DRY configurations of env vars that target 3 different axes of configuration:
1. the application - eg 'api' or 'background-worker'
2. the environment - eg 'dev' or 'prod' (but allow arbitrary env names)
3. the target - e.g. 'docker' or 'kubernetes' or 'local'

Additionally there will be `shared.yml` files to store shared config vars

# Config value storage
The files should be structured like this:
```
$PUFF_ROOT/
    
    base/
        shared.yml - truly global
        api.yml - base config for api app
        background-worker.yml
        some-other-app.yml
    dev/
        shared.yml - shared between all apps for dev env, e.g. a database url
        api.yml
        background-worker.yml
    prod/
        shared.yml
        api.yml
        background-worker.yml
        some-other-app.yml
    target-overrides/
        docker/
            shared.yml
            api.yml - docker-specific config
```

# Merge order

From lowest to highest precedence:

1. global shared
2. base app-specific
3. env shared
4. env app-specific
5. target shared
6. target app-specific


# Output

This tool will support the following output formats:
1. k8s secrets (plain text by default, base64 as an option), will need to specify the secret name
2. .env files - nested values should be converted to JSON
3. json
4. yaml

it will be called somewhat like `puff generate --app api --env dev --target local --format env --output some_file_name.env`


# Templating

The tool should support templated values from higher in the precedence chain, or previously in the same file:

base/shared.yml:
```
PAGE_TITLE: My Cool App
```

dev/frontend.yml:
```
TITLE_TEXT: ${PAGE_TITLE} (staging)
```

would result in generating 
```
PAGE_TITLE="My Cool App"
TITLE="My Cool App (staging)"
```

if the user called `puff generate --app frontend --env dev --target local --format env`

If a user wants to have a value available for templating, but does NOT want that value to propagate to the final generated result, they can prefix it with an underscore:

base/shared.yml:
```
_PAGE_TITLE: My Cool App
```

dev/frontend.yml:
```
TITLE_TEXT: ${_PAGE_TITLE} (staging)
```

would result in generating 
```
TITLE="My Cool App (staging)"
```

# Commands

* init - setup config dir including .sops.yaml
* keys - parent command for key management, supports an --env flag to only update files in specific envs
  * add - add an age key and re-encrypt all files with said key, supports a --comment flag to include a comment like "Bob's new laptop key"
  * rm - remove an age key and re-encrypt all files
  * list - list all keys with their comments, including what envs they have access to
* get - get a config value for a specified app/env/target (this should be the final value)
* set - set a config value for a specified app/env/target. if no app or env is specified, add it to `base/shared.yml`. If only app is specified, add it to `base/app-name.yml`. If only env is specified, add to `env-name/shared.yml`. If target is specified, add to either `target-overrides/target-name/shared.yml` or `target-overrides/target-name/app-name.yml` if app is specified
* generate - generate the full config for the specified app/env/target, in the specified format. If an output file is specified, write it to that file. If not, send to STDOUT

# Testing
Aside from the usual unit tests, this project will have real integration tests that test the full functionality end-to-end by invoking puff's binary from the shell. Ensure your tests are DRY as well!

# Libraries
use https://github.com/urfave/cli for implementing the CLI
use https://github.com/fatih/color to make the terminal output pleasing
use https://github.com/spf13/viper to configure puff itself, as well as for creating a test consuming application for puff's output
