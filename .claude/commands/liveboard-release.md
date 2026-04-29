---
description: Cut a LiveBoard release — bump versions, commit, tag, push
argument-hint: patch | minor | major
allowed-tools: [Read, Edit, Bash]
---

# LiveBoard release

Drive the full LiveBoard release flow from a clean `main` branch. Pushing the tag triggers `.github/workflows/release.yml` (goreleaser) and `.github/workflows/release-desktop.yml` (macOS bundle + Homebrew cask).

User invoked with: `$ARGUMENTS`

## Steps

Execute these in order. Abort with a clear message on any failure — do **not** attempt to auto-fix via stash, reset, or force.

### 1. Validate argument

`$ARGUMENTS` must be exactly `patch`, `minor`, or `major`. Anything else (including empty): abort and print:

```
Usage: /liveboard-release patch | minor | major
```

### 2. Verify clean state

Run all of these, abort on any failure:

- `git rev-parse --abbrev-ref HEAD` → must equal `main`.
- `git status --porcelain` → must be empty (no staged, unstaged, or untracked files).
- `git fetch origin main`
- `git rev-list --count HEAD..origin/main` → must equal `0` (local must not be behind).

### 3. Read current version

Read `web/shell/package.json` and extract the `version` field. This is the single source of truth — the other two `package.json` files must already match it.

Sanity-check by reading `web/shared/package.json` and `web/renderer/default/package.json` and confirming all three versions are identical. If they diverge, abort and tell the user.

### 4. Compute new version

Apply semver to `$ARGUMENTS`:

- `patch`: bump the third component, e.g. `0.20.3` → `0.20.4`
- `minor`: bump the second, reset third, e.g. `0.20.3` → `0.21.0`
- `major`: bump the first, reset second and third, e.g. `0.20.3` → `1.0.0`

Tell the user: `Releasing <old> → <new>`.

### 5. Pre-flight checks

Run, in order, aborting on non-zero exit:

- `make lint`
- `make build`

### 6. Bump version in three package.json files

Use the `Edit` tool with the exact string `"version": "<old>"` → `"version": "<new>"` in each:

- `web/shell/package.json`
- `web/shared/package.json`
- `web/renderer/default/package.json`

### 7. Build the npm artifacts and detect renderer asset hashes

The landing page references content-hashed Vite output (`index-<hash>.js` and `index-<hash>.css`) from `@wriven/liveboard-renderer-default`. Vite emits a fresh hash on every build, so each release has different filenames — they must be detected from the just-built `dist-npm/` and patched into the landing-page HTML, otherwise CDN URLs will 404.

Run:

```
make npm-build
```

Then capture the new asset filenames (each glob matches exactly one file):

```
NEW_JS=$(basename web/renderer/default/dist-npm/assets/index-*.js)
NEW_CSS=$(basename web/renderer/default/dist-npm/assets/index-*.css)
```

If either glob matches zero or more than one file, abort and tell the user — the renderer build did not produce the expected entry chunk.

Also capture the *current* (pre-edit) hashes from the landing page:

```
OLD_JS=$(grep -oE 'index-[A-Za-z0-9_-]+\.js' docs/landing-page/index.html | head -1)
OLD_CSS=$(grep -oE 'index-[A-Za-z0-9_-]+\.css' docs/landing-page/index.html | head -1)
```

### 8. Update landing-page version + asset hashes

The landing page is split across two files:

- `docs/landing-page/index.html` — CDN URLs for both the displayed code example and the live demo iframe shell
- `docs/landing-page/renderer.html` — same-origin renderer document loaded into the iframe

Apply the following `Edit` calls with `replace_all: true` to **both** files:

1. `@<old-version>/` → `@<new-version>/`
2. `<OLD_JS>` → `<NEW_JS>`
3. `<OLD_CSS>` → `<NEW_CSS>`

Verify with grep: there should be zero remaining occurrences of `@<old-version>/`, `<OLD_JS>`, or `<OLD_CSS>` across both files.

### 9. Show diff and confirm

Run `git diff --stat` and `git diff` (the latter scoped or summarized — full diff if small, summary if large). Tell the user what's about to be committed, tagged, and pushed, then **ask for explicit confirmation** before continuing. Pushing `main` and a release tag is the irreversible step — do not skip this gate.

Note: `make npm-build` writes into `web/renderer/default/dist-npm/` and `web/shell/dist-npm/`. Both directories are gitignored, so they will not appear in `git status` — but verify nothing unexpected is staged.

### 10. Commit

Stage the five files **by name** (never `git add -A` / `git add .`):

```
git add web/shell/package.json web/shared/package.json web/renderer/default/package.json docs/landing-page/index.html docs/landing-page/renderer.html
```

Commit with a HEREDOC body, message exactly:

```
chore: bump npm packages to <new>, update landing page CDN refs

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
```

Never use `--no-verify` or `--amend`. If a hook fails, fix the underlying issue and create a new commit.

### 11. Tag

```
git tag v<new>
```

(No `-s`, no `-a` — match the existing tag style; `git tag --list 'v*'` confirms simple lightweight tags.)

### 12. Push

After the user confirmed in step 9, push commit then tag:

```
git push origin main
git push origin v<new>
```

Then print the release URL so the user can watch CI:

```
https://github.com/<owner>/<repo>/releases/tag/v<new>
```

(Resolve `<owner>/<repo>` from `git remote get-url origin`.)

### 13. Done

Tell the user:

- CI will publish the GitHub release, macOS desktop zip, and Homebrew cask update within ~10 minutes.
- npm publish is a separate step (`make npm-publish`) — do **not** run it unless the user explicitly asks.

## Guardrails

- Never `--no-verify`, never `--force`, never `--amend` published commits.
- Never `git add -A` / `git add .` — stage by filename.
- Never auto-resolve a dirty tree — abort and let the user clean up.
- The confirmation gate before push (step 9) is mandatory.
