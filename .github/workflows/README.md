# GitHub Workflows

## Overview

Following workflows are implemented in the repository.
[SARIF](https://docs.github.com/en/code-security/code-scanning/integrating-with-code-scanning/sarif-support-for-code-scanning) is used to store the results for an analysis of code scanning tools in the Security tab of the repository.

| Workflow                                         | Jobs                            | Trigger                                                       | SARIF upload | Description                                                                                     |
| :----------------------------------------------- | :------------------------------ | :------------------------------------------------------------ | :----------- | ----------------------------------------------------------------------------------------------- |
| [codeql.yml](./codeql.yml)                       | `analyze`                       | push/pr to `main`, cron: `00 13 * * 1`                        | yes          | Semantic code analysis                                                                          |
| [dependency-review.yml](./dependency-review.yml) | `dependency-review`             | pr to `main`                                                  | -            | Check pull request for vulnerabilities in dependencies or invalid licenses are being introduced |
| [golangci-lint.yml](./golangci-lint.yml)         | `lint`                          | push/pr on `*`                                                | -            | Lint Go Code                                                                                    |
| [gosec.yml](./gosec.yml)                         | `analyze`                       | push/pr on `*`                                                | -            | Inspects source code for security problems in Go code                                           |
| [osv-scan.yml](./osv-scan.yml)                   | `analyze`                       | push/pr to `main`, cron: `30 13 * * 1`                        | yes          | Scanning for vulnerabilites in dependencies                                                     |
| [release.yml](./release.yml)                     | see [release chapter](#release) | push tag `v*`                                                 | -            | Create release with go binaries and docker container                                            |
| [scorecard.yml](./scorecard.yml)                 | `analyze`                       | push to `main`, cron: `00 14 * * 1`, change branch protection | yes          | Create OpenSSF analysis and create project score                                                |

## CodeQL

Action: https://github.com/github/codeql-action

[CodeQL](https://codeql.github.com/) is a semantic code analysis engine that can find security vulnerabilities in codebases. The workflow displays security alerts in the repository's Security tab or in pull requests.

## Dependency Review

Action: https://github.com/actions/dependency-review-action

This action scans the dependency manifest files that change as part of a pull request, revealing known-vulnerable versions of the packages declared or updated in the PR. Pull requests that introduce known-vulnerable packages will be blocked from merging.
It also allows you to define a list of licenses that are allowed or disallowed in the project, and will check if the PR introduces a dependency with a disallowed license.
It also checks the OpenSSF scorecard for all dependencies and allows to warn if a dependency has a low score.

More information can be found in the [GitHub documentation](https://docs.github.com/en/code-security/supply-chain-security/understanding-your-software-supply-chain/about-dependency-review)

## GolangCI-Lint

Action: https://github.com/golangci/golangci-lint-action

[GolangCI-Lint](https://golangci-lint.run/) is a fast Go linters runner. It runs linters in parallel, uses caching, and works on Linux, macOS, and Windows. The workflow runs the linters on the Go code in the repository.

## Gosec

Action: https://github.com/securego/gosec

[Gosec](https://securego.io/) is a security tool that performs static code analysis of Go code. The workflow scans the Go code in the repository for security issues.

## OSV-Scan

Action: https://github.com/google/osv-scanner-action

[OSV-Scan](https://osv.dev/) is a vulnerability database and triage infrastructure for open-source projects. The [OSV-Scanner](https://google.github.io/osv-scanner/) finds vulnerabilities in dependencies of an project and uploads the results to the Security tab of the repository.

## Release

The release workflow includes multiple jobs to create a release of the project. Following jobs are implemented:

| Job                               | GitHub Action                                                                                                                    | Description                                                                        |
| :-------------------------------- | :------------------------------------------------------------------------------------------------------------------------------- | :--------------------------------------------------------------------------------- |
| `goreleaser`                      | [goreleaser-action](https://github.com/goreleaser/goreleaser-action)                                                             | Creates the go archives & checksums file                                           |
| `ko-publish`                      | [publish-image action](../actions/publish-image/action.yaml)                                                                     | Create the container images & SBOMs, sign images and upload to the GitHub registry |
| `binary-provenance`               | [generator_generic_slsa3](https://github.com/slsa-framework/slsa-github-generator/blob/main/internal/builders/generic/README.md) | Generate provenance for all release artifacts (go archives & SBOMs)                |
| `image-provenance`                | [generator_container_slsa3](https://github.com/slsa-framework/slsa-github-generator/tree/main/internal/builders/container)       | Generates provenance for the container images                                      |
| `verification-with-slsa-verifier` | -                                                                                                                                | Verifying the cryptographic signatures on provenance for all binary releases       |
| `verification-with-cosign`        | -                                                                                                                                | Verifying the cryptographic signatures on provenance for the container image       |

### Go Release

This repository uses [goreleaser](https://goreleaser.com/) to create all the release artifacts. GoReleaser can build and release Go binaries for multiple platforms, create archives/container images/SBOMs and more. All the configuration for the release is stored in the file [.goreleaser.yml](./../../.goreleaser.yml).
For all the release artifacts (`*.tar.gz`, `*.zip`, `*.sbom.json`), provenance is generated using the [SLSA Generic Generator](https://github.com/slsa-framework/slsa-github-generator/blob/main/internal/builders/generic/README.md). The provenance file is uploaded to the release assets and can be verified using the `slsa-verifier` tool (see [Release Verification](./../../SECURITY.md#release-verification)).

### Container Release

The multi-arch container images are built using [ko](https://ko.build/) in the [publish-image](../actions/publish-image/action.yaml) action and uploaded to the GitHub Container Registry. The docker image provenance is generated using the [SLSA Container Generator](https://github.com/slsa-framework/slsa-github-generator/tree/main/internal/builders/container) and uploaded to the registry. The provenance can be verified using the `slsa-verifier` or `cosign` tool (see [Release Verification](./../../SECURITY.md#release-verification)).

**Credits**: The [publish-image](../actions/publish-image/action.yaml) action is from [Kyverno](https://github.com/kyverno/kyverno).

### Container SBOM

[ko](https://ko.build/features/sboms/) only generates a "minimal" SBOM for the container images (see [comment in GitHub Issue](https://github.com/ko-build/ko/pull/587#issuecomment-1034926085)) and lacks some information (e.g. Licensing information or the `version` field which is set to `devel` instead of the actual version).

To generate a complete SBOM for the container images, the [go-gomod-generate-sbom](https://github.com/CycloneDX/gh-gomod-generate-sbom) action is used instead.

The SBOMs of the container images are uploaded to a separate package registry (see [SBOM](./../../SECURITY.md#sbom) for more information).

## Scorecards

Action: https://github.com/ossf/scorecard-action

[Scorecards](https://github.com/ossf/scorecard) is a tool that provides a security score for open-source projects. The workflow runs the scorecard on the repository and uploads the results to the Security tab of the repository. There is also a report on the OpenSSF website, the link is available in the README file by clicking on the OpenSSF Scorecard badge.

[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/natrontech/pbs-exporter/badge)](https://securityscorecards.dev/viewer/?uri=github.com/natrontech/pbs-exporter)
