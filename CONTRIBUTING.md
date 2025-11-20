# Contributing to Puff

Thank you for your interest in contributing to Puff! This document provides guidelines and instructions for contributing.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/puff.git`
3. Add the upstream remote: `git remote add upstream https://github.com/teamcurri/puff.git`
4. Create a new branch for your feature: `git checkout -b feature/my-feature`

## Development Setup

### Prerequisites

- Go 1.21 or later
- Make (optional, but recommended)

### Building

```bash
# Build the binary
make build

# Or manually
go build -o bin/puff ./cmd/puff
```

### Running Tests

```bash
# Run all tests
make test

# Run only unit tests
make test-unit

# Run only integration tests
make test-integration

# Generate coverage report
make coverage
```

## Project Structure

```
puff/
├── cmd/puff/           # Main application entry point
├── pkg/
│   ├── config/         # Configuration loading and merging
│   ├── templating/     # Variable template resolution
│   ├── output/         # Output format generators
│   ├── keys/           # SOPS key management
│   └── commands/       # CLI command implementations
├── test/               # Integration tests
├── examples/           # Example configurations
└── docs/               # Additional documentation
```

## Coding Standards

### Code Style

- Follow standard Go conventions
- Run `go fmt` before committing
- Run `go vet` to catch common errors
- Use meaningful variable and function names
- Add comments for exported functions and types

```bash
# Format code
make fmt

# Run linters
make lint
```

### Testing

- Write unit tests for all new functionality
- Maintain or improve test coverage
- Integration tests should cover end-to-end scenarios
- Test both success and error cases

### Commit Messages

Write clear, descriptive commit messages:

```
Short (50 chars or less) summary

More detailed explanatory text, if necessary. Wrap it to about 72
characters or so. In some contexts, the first line is treated as the
subject of an email and the rest of the text as the body.

- Bullet points are okay
- Use present tense ("Add feature" not "Added feature")
- Reference issues and pull requests liberally
```

## Making Changes

1. **Create a feature branch** from `main`:
   ```bash
   git checkout -b feature/my-feature
   ```

2. **Make your changes** following the coding standards

3. **Write tests** for your changes

4. **Run tests** to ensure everything works:
   ```bash
   make test
   ```

5. **Commit your changes**:
   ```bash
   git add .
   git commit -m "Add feature: description"
   ```

6. **Push to your fork**:
   ```bash
   git push origin feature/my-feature
   ```

7. **Create a Pull Request** from your fork to the main repository

## Pull Request Process

1. Update the README.md with details of changes if applicable
2. Update documentation if you're changing functionality
3. Add tests for any new features
4. Ensure all tests pass
5. Reference any relevant issues in the PR description

### PR Checklist

- [ ] Tests pass locally (`make test`)
- [ ] Code is formatted (`make fmt`)
- [ ] Linters pass (`make lint`)
- [ ] Documentation is updated
- [ ] Commit messages are clear and descriptive
- [ ] Changes are focused and atomic

## Areas for Contribution

### High Priority

- **Full SOPS Integration**: Complete implementation of key management commands
- **Performance Optimization**: Improve config loading and merging performance
- **Error Messages**: Better error messages with suggestions
- **CLI Improvements**: Better help text and examples

### Features to Consider

- Support for additional encryption backends (Vault, AWS KMS, etc.)
- Configuration validation and schema support
- Config diff command to compare environments
- Import/export from other config management tools
- Shell completion scripts (bash, zsh, fish)
- Configuration templates for common setups
- Watch mode for live reloading during development
- Dry-run mode for generate commands
- Backup and restore functionality

### Documentation

- More examples for different use cases
- Video tutorials
- Migration guides from other tools
- Best practices guide
- Troubleshooting guide

## Reporting Bugs

When reporting bugs, please include:

1. **Description**: Clear description of the bug
2. **Steps to Reproduce**: Minimal steps to reproduce the issue
3. **Expected Behavior**: What you expected to happen
4. **Actual Behavior**: What actually happened
5. **Environment**: OS, Go version, Puff version
6. **Logs**: Relevant error messages or logs

Example:

```markdown
### Bug: Generate command fails with nested templates

**Description**: The generate command crashes when using nested template variables.

**Steps to Reproduce**:
1. Create config with nested templates (see example below)
2. Run: `puff generate -a api -e dev -f env`
3. Error occurs

**Config files**:
base/shared.yml:
```yaml
_BASE: value
_DERIVED: ${_BASE}/path
FINAL: ${_DERIVED}/file
```

**Expected**: Output should show `FINAL=value/path/file`

**Actual**: Error: "circular dependency detected"

**Environment**:
- OS: macOS 14.0
- Go version: 1.21.0
- Puff version: v0.1.0
```

## Feature Requests

For feature requests, please provide:

1. **Use Case**: Describe the problem you're trying to solve
2. **Proposed Solution**: How you envision the feature working
3. **Alternatives**: Other ways you've considered solving this
4. **Examples**: Concrete examples of how the feature would be used

## Code Review Process

- All submissions require review before merging
- Maintainers will review PRs within a few days
- Address review feedback by pushing new commits
- Once approved, maintainers will merge the PR

## Questions?

Feel free to:
- Open an issue for questions
- Start a discussion in GitHub Discussions
- Reach out to the maintainers

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

Thank you for contributing to Puff!
