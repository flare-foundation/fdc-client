# Contributing

Thank you for considering improving out source code.
All contributions are welcome.

## Issues

_Sensitive security-related issues should be reported to any of [codeowners](/CODEOWNERS)._

To share ideas, considerations, or concerned open an issue.
Before filing an issue make sure the issue has not been already raised.
In the issue, answer the following questions:

- What is the issue?
- Why it is an issue?
- How do you propose to change it?

### Pull request

Before opening a pull request open an issue on why the request is needed.

To contribute: fork the repo, make your improvements, commit and open a pull request.
The maintainers will review the request.

The request must:

- Reference the relevant issue,
- Follow standard golang guidelines,
- Be well documented,
- Be well tested,
- Compile,
- Pass all the tests,
- Pass all the linters,
- Be based on opened against `main` branch.

## Setting the environment

Make sure you are using a version of go higher or equal to the one specified in go.mod.

Get all the dependencies:

```bash
go mod tidy
```

Run all tests:

```bash
go test ./...
```

Generate a coverage report:

```bash
$ ./gencover.sh
```

Run linters (make sure you have [golangci-lint](https://golangci-lint.run/) installed) with

```bash
golangci-lint run
```
