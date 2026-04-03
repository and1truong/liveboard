# Release Desktop Workflow

Add a GitHub Actions workflow that builds and publishes the macOS desktop `.app` bundle on tag push.

## Context

- Desktop app exists: Wails v2 in `cmd/liveboard-desktop/`, `.app` bundling via Makefile
- `make bundle-desktop-release` builds universal (arm64+amd64) binary, assembles `.app`, zips it
- `make release-desktop` uploads zip to GitHub release + updates Homebrew cask
- Current `release.yml` runs on `ubuntu-latest` (GoReleaser, CLI only) — can't build desktop (needs macOS, CGO, system frameworks)

## Design

### New file: `.github/workflows/release-desktop.yml`

**Trigger**: `push.tags: ["v*"]` (same as `release.yml`)

**Runner**: `macos-latest`

**Permissions**: `contents: write`

**Steps**:

1. **Checkout** — `actions/checkout` with `fetch-depth: 0` (needed for `git describe --tags`)
2. **Setup Go** — `actions/setup-go` with `go-version-file: go.mod`, cache enabled
3. **Install Tailwind** — download `tailwindcss-macos-arm64` binary (macos-latest is arm64)
4. **Wait for release** — poll `gh release view $TAG` every 15s, max 5 min. GoReleaser in `release.yml` creates the release; this job needs it to exist before uploading. Fail if timeout.
5. **Build** — `make bundle-desktop-release`
6. **Upload to release** — `gh release upload $TAG LiveBoard-*-macos-universal.zip --clobber`
7. **Update Homebrew cask** — `bash scripts/update-desktop-cask.sh $VERSION`

**Secrets**: `HOMEBREW_TAP_TOKEN` (already configured for CLI release)

**Environment variables**:
- `GITHUB_TOKEN` for `gh` CLI (release upload, release view polling)
- `HOMEBREW_TAP_TOKEN` for cask update step — must be set as `GH_TOKEN` when running the script so `gh repo clone` and `git push` authenticate against the tap repo (the default `GITHUB_TOKEN` lacks cross-repo push access)

### Race condition handling

Both `release.yml` and `release-desktop.yml` trigger on the same `v*` tag push. GoReleaser creates the GitHub release in `release.yml`. The desktop workflow must wait for that release to exist before uploading its artifact. A simple retry loop on `gh release view` handles this.

### Not in scope

- Code signing / notarization (deferred)
- DMG packaging (zip is sufficient for now)
- Windows/Linux desktop builds (not applicable — Wails app uses macOS frameworks)

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Workflow structure | Separate `release-desktop.yml` | Isolate macOS concerns from CLI release |
| Build approach | Reuse Makefile targets | Single source of truth, tested locally |
| Cask update | Automated in CI | Same `HOMEBREW_TAP_TOKEN` secret already available |
| Signing | Deferred | Ship unsigned first, add later |
| Release sync | Poll `gh release view` | Simple, no cross-workflow dependency needed |
