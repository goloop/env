# Contributing

We love your input! We want to make contributing to this repository as simple and transparent as possible, be it:
- Reporting a bug
- Discussing the current state of the code
- Submitting a fix
- Proposing new features

## Code of Conduct

Please note that this project has a Code of Conduct (see CODE_OF_CONDUCT.md). By participating in this project, you agree to abide by its terms. We are committed to providing a welcoming and inclusive environment for all contributors.

## Development Process

1. Fork the repo and create your branch from `master`.
2. Make your changes.
3. If you've added code, add tests.
4. Ensure the test suite passes.
5. Update benchmarks if necessary.
6. Create a pull request.

## Testing Your Changes

### Running Tests

Run all tests to ensure nothing breaks:
```bash
go test ./...
```

### Test Coverage

Check test coverage for your changes:
```bash
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

If your changes aren't covered by existing tests, please add appropriate test cases. We aim to maintain high test coverage to ensure code reliability.

### Performance Testing

If your changes might affect performance, you should compare benchmarks:

1. First, run benchmarks on the `master` branch:
```bash
git checkout master
go test -bench=. -benchmem > /tmp/old_benchmarks.txt
```

2. Then, switch to your branch and run the same benchmarks:
```bash
git checkout your-branch
go test -bench=. -benchmem > /tmp/new_benchmarks.txt
```

3. Install benchstat if you haven't already:
```bash
go install golang.org/x/perf/cmd/benchstat@latest
```

4. Compare results:
```bash
benchstat /tmp/old_benchmarks.txt /tmp/new_benchmarks.txt
```

The `benchstat` tool will show statistical analysis of the changes:
- "old" column shows existing benchmark results
- "new" column shows your changes
- "delta" column shows the difference (negative means faster)
- Statistical significance is indicated (p-value < 0.05)

Example output:
```
name        old time/op  new time/op  delta
JSONSmall-8  387ns ± 1%  350ns ± 2%  -9.56%  (p=0.008)
```

This shows your changes made the JSONSmall benchmark 9.56% faster, and this improvement is statistically significant.

Note: While absolute performance numbers may vary between different machines, running benchmarks on the same machine for both branches allows for meaningful relative comparisons.

### Code Style

- Follow standard Go code style and conventions
- Use `gofmt` to format your code
- Run `golint` and `go vet` to catch common issues
- Add comments for exported functions and types
- Update documentation if necessary

### Documentation

If you've added or modified functionality:
1. Update relevant documentation in code comments
2. Update README.md if necessary
3. Update examples if applicable

### Commit Messages

- Use clear and meaningful commit messages
- Start with a verb in imperative mood (Add, Fix, Change, etc.)
- Keep the first line under 70 characters
- Reference issues and pull requests where appropriate

Example:
```
Add gzip compression support for JSON responses

- Implement transparent compression for large responses
- Add compression threshold configuration
- Update documentation with compression examples
- Add benchmarks for compressed responses

Fixes #123
```

## Pull Request Process

1. Update the README.md with details of changes if applicable
2. Update the documentation with details of any interface changes
3. Add your changes to the CHANGELOG.md under "Unreleased"
4. The PR will be merged once you have the sign-off of at least one maintainer

## License

By contributing, you agree that your contributions will be licensed under the same license as the original project.

## Questions?

Feel free to open an issue for any questions about contributing!
