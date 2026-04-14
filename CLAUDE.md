# CLAUDE.md

## Updating Go version and dependencies

When updating to a new Go version or upgrading dependencies, touch all of the following locations:

### 1. Update Go modules

```bash
# Update all dependencies to latest versions
make go-update   # runs: go get -u ./... && go mod tidy -compat=<major.minor>
```

Or manually:

```bash
go get -u ./...
go mod tidy
```

### 2. Update the Go version directive in go.mod

Edit [go.mod](go.mod) and bump the `go` directive:

```
go 1.26.2
```

### 3. Update hardcoded GOTOOLCHAIN in workflows

Two workflows hardcode the toolchain version and must be updated manually:

- [.github/workflows/codeql.yml](.github/workflows/codeql.yml) — `GOTOOLCHAIN: "go1.26.2"`
- [.github/workflows/gosec.yml](.github/workflows/gosec.yml) — `GOTOOLCHAIN: "go1.26.2"`

The following workflows use `go-version-file: 'go.mod'` and pick up the version automatically — no changes needed:

- [.github/workflows/golangci-lint.yml](.github/workflows/golangci-lint.yml)
- [.github/workflows/release.yml](.github/workflows/release.yml)

### 4. Update `-compat` flag in Makefile (major/minor version bumps only)

[Makefile](Makefile) line 15 has a hardcoded `-compat` flag:

```makefile
go mod tidy -compat=1.26
```

Update this when the `major.minor` version changes (not needed for patch-only bumps).

### 5. Update ko version

[Makefile](Makefile) hardcodes the ko version:

```makefile
KO_VERSION  = v0.18.1
```

Check the latest release and update the version:

```bash
gh release view --repo google/ko --json tagName -q '.tagName'
```

Then update `KO_VERSION` in [Makefile](Makefile) accordingly.

### 6. Update GitHub Actions versions

All workflow files under [.github/workflows/](.github/workflows/) pin actions by commit SHA with a tag comment, e.g.:

```yaml
uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2
```

To update, get the latest tag and its commit SHA for each action:

```bash
# Get latest tag
gh release view --repo <owner>/<repo> --json tagName -q '.tagName'

# Get commit SHA for that tag
gh api repos/<owner>/<repo>/commits/<tag> --jq '.sha'
```

Then update the SHA and the tag comment in the workflow file.

**Exception — `slsa-framework/slsa-github-generator`** must be referenced by tag, not SHA (see [upstream docs](https://github.com/slsa-framework/slsa-github-generator/?tab=readme-ov-file#referencing-slsa-builders-and-generators)):

```yaml
uses: slsa-framework/slsa-github-generator/.github/workflows/generator_generic_slsa3.yml@v2.1.0
```

Actions used across the workflows:

| Action | Workflow(s) |
|---|---|
| `actions/checkout` | all |
| `actions/dependency-review-action` | dependency-review.yml |
| `actions/setup-go` | release.yml, golangci-lint.yml |
| `actions/upload-artifact` | scorecard.yml |
| `anchore/sbom-action` | release.yml |
| `creekorful/goreportcard-action` | release.yml |
| `docker/login-action` | release.yml |
| `github/codeql-action` | codeql.yml, scorecard.yml |
| `golangci/golangci-lint-action` | golangci-lint.yml — also update `version:` param to match `rev` in [.pre-commit-config.yaml](.pre-commit-config.yaml) |
| `google/osv-scanner-action` | osv-scan.yml |
| `goreleaser/goreleaser-action` | release.yml |
| `ossf/scorecard-action` | scorecard.yml |
| `securego/gosec` | gosec.yml |
| `sigstore/cosign-installer` | release.yml, release-verification.yml |
| `slsa-framework/slsa-github-generator` | release.yml (**tag only**) |
| `slsa-framework/slsa-verifier` | release-verification.yml |

### 7. Update pre-commit hooks

[.pre-commit-config.yaml](.pre-commit-config.yaml) pins the `rev` of each hook repository. Update all revisions to their latest tags:

```bash
prek auto-update
```

This updates the `rev` fields for all four repos in [.pre-commit-config.yaml](.pre-commit-config.yaml):
- `pre-commit/pre-commit-hooks`
- `gitleaks/gitleaks`
- `dnephin/pre-commit-golang`
- `golangci/golangci-lint`

### 8. Verify

```bash
go build ./...
go test ./...
```
