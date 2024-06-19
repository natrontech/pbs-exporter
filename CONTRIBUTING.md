# Contributing

All contributions are welcome! If you find a bug or have a feature request, please open an issue or submit a pull request.

Please note that we have a [Code of Conduct](./CODE_OF_CONDUCT.md), please follow it in all your interactions with the project.

## How to Contribute

You can make a contribution by following these steps:

  1. Fork this repository, and develop your changes on that fork.
  2. Commit your changes
  3. Submit a [pull request](#pull-requests) from your fork to this project.

Before you start, read through the requirements below.  

### Commits

Please make your commit messages meaningful. We recommend creating commit messages according to [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/).

### Commit Signature Verification

Each commit's signature must be verified.

  * [About commit signature verification](https://docs.github.com/en/free-pro-team@latest/github/authenticating-to-github/about-commit-signature-verification)

### Pull Requests

All contributions, including those made by project members, need to be reviewed. We use GitHub pull requests for this purpose. See [GitHub Help](https://help.github.com/articles/about-pull-requests/) for more information on how to use pull requests. See the requirements above for PR on this project.

### Major new features

If a major new feature is added, there should be new tests for it. If there are no tests, the PR will not be merged.

### Versioning

Versions follow [Semantic Versioning](https://semver.org/) terminology and are expressed as `x.y.z`:

- where `x` is the major version
- `y` is the minor version
- and `z` is the patch version

## Code convention

## Pre-Commit

Please install [pre-commit](https://pre-commit.com/) to enforce some pre-commit checks.
After cloning the repository, you will need to install the hook script manually:

```bash
pre-commit install
```

## golint

We will check the code against [golangci-lint](https://github.com/golangci/golangci-lint) to enforce some code conventions. It runs in the CI pipeline and in pre-commit. You can also run it manually:

```bash
golangci-lint run
```

## goimports

We have a pre-commit hook that runs goimports to update the Go import lines, adding missing ones and removing unreferenced ones. You can install it with:

```bash
go install golang.org/x/tools/cmd/goimports@latest
```
