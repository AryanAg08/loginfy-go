# Contributing to Lognify.go

Thank you for your interest in contributing to Lognify.go! This guide will help you get started.

## How to Contribute

1. **Fork** the repository on GitHub.
2. **Create a branch** from `main` for your feature or bug fix.
3. **Make your changes**, following the guidelines below.
4. **Submit a pull request** back to the `main` branch.

## Development Setup

```bash
# Clone your fork
git clone https://github.com/<your-username>/Lognify.go.git
cd Lognify.go

# Install dependencies
go mod download

# Run tests
go test ./...

# Run tests with race detector
go test -v -race -count=1 ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## Code Style Guidelines

- Follow standard Go conventions and idioms.
- Run `gofmt` on all code before committing — the CI will reject unformatted code.
- Run `go vet ./...` to catch common issues.
- Keep functions focused and well-named; avoid unnecessary comments on self-explanatory code.
- Export only what needs to be part of the public API.

## PR Process

1. Ensure all tests pass locally before opening a PR.
2. Write a clear PR title and description explaining **what** changed and **why**.
3. Reference any related issues (e.g., `Closes #42`).
4. Keep PRs small and focused — one logical change per PR.
5. CI will automatically run tests, linting, and security checks on your PR.
6. Address any review feedback promptly.

## Testing Guidelines

- All new features and bug fixes **must** include tests.
- Run the full test suite before submitting: `go test -v -race ./...`
- Aim for meaningful coverage — focus on edge cases and error paths, not just the happy path.

## Adding Tests for New Features

### Where Tests Live

Tests are located in the `tests/` directory at the project root. Place new test files there following existing conventions.

### Test Naming Conventions

- Test files should end with `_test.go`.
- Test functions must start with `Test` followed by a descriptive name:
  ```go
  func TestEmailPasswordStrategy_ValidCredentials(t *testing.T) { ... }
  ```
- Use table-driven tests where appropriate for covering multiple cases.

### Adding Tests for New Strategies

If you add a new authentication strategy under `strategies/`:

1. Create a test file in `tests/` (e.g., `tests/oauth_strategy_test.go`).
2. Test the strategy's `Authenticate` and any other public methods.
3. Cover success, failure, and edge-case scenarios.

### Adding Tests for New Storage Adapters

If you add a new storage adapter under `storage/`:

1. Create a test file in `tests/` (e.g., `tests/redis_storage_test.go`).
2. Test CRUD operations: create, read, update, and delete.
3. Test error handling for connection failures or invalid data.

### Adding Tests for New Middleware

If you add new middleware under `middleware/`:

1. Create a test file in `tests/` (e.g., `tests/rate_limit_middleware_test.go`).
2. Use `net/http/httptest` to simulate HTTP requests.
3. Test that the middleware correctly allows, modifies, or rejects requests.

### Running Specific Tests

```bash
# Run a single test function
go test -v -run TestEmailPasswordStrategy_ValidCredentials ./tests/...

# Run all tests in a specific package
go test -v ./storage/memory/...

# Run all tests
go test -v ./...
```

### CI Integration

CI will automatically run the full test suite on every pull request. You can see the workflow configuration in `.github/workflows/ci.yml`. Make sure your tests pass locally before pushing to avoid CI failures.
