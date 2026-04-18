| name | description | user_invocable |
| --- | --- | --- |
| commit | Create git commits using Conventional Commits format. Use when user asks to commit, save changes, or says /commit. Trigger on 'conventional commit', 'commit message', or stage+commit requests. Handles diff analysis, auto-detecting type/scope, generating messages, optionally splitting into multiple logical commits. | true |

## Conventional Commits Skill

Create professional git commits following Conventional Commits spec.

## Step 1: Gather context

Run in parallel:

1. `git status` — see changes (never use `-uall`)
2. `git diff --staged` — staged changes
3. `git diff` — unstaged changes
4. `git log --oneline -5` — recent commit style

## Step 2: Analyze changes and decide commit strategy

Look at all changes holistically. Determine:

- __Multiple logical change groups?__ E.g. new feature in one package + bug fix in another, or migrations + service + handler as distinct units. If so, ask user about splitting. Suggest split when changes touch unrelated areas — don't over-split. Related changes (service + handler + tests) belong in one commit.
- __Which files to stage?__ If nothing staged, figure out which files belong together. Never stage secrets (`.env`, credentials, keys). Prefer `git add <specific files>` over `git add -A` or `git add .`.

## Step 3: Determine type, scope, and description

### Type

Pick based on nature of change, not files changed:

| Type | When |
| --- | --- |
| `feat` | New feature/capability |
| `fix` | Bug fix — broken → works |
| `refactor` | Code restructuring, no behavior change |
| `docs` | Documentation only |
| `chore` | Maintenance: deps, configs, build, tooling |
| `test` | Adding/updating tests only |
| `ci` | CI/CD pipeline changes |
| `perf` | Performance improvement |
| `style` | Formatting, whitespace (no logic change) |

Feature + its tests → use `feat`. Type reflects primary intent.

### Scope

Derive from primary codebase area. Short, recognizable names:

- Go: package/directory (e.g., `auth`, `billing`, `middleware`)
- JS/TS: component/module (e.g., `dashboard`, `api`, `hooks`)
- Config/infra: system (e.g., `docker`, `ci`, `deps`)
- Migrations: `db` or `migrations`
- Spans many areas w/ no clear primary → omit scope

### Description

- Lowercase, imperative ("add", "fix", "remove" — not "added", "fixes")
- No period at end
- Under 72 characters
- Describe _what_, not _how_

Good: `feat(auth): add JWT refresh token rotation`
Bad: `feat(auth): Added JWT refresh token rotation.`

### Body (optional)

Include only if changes need explanation — "why" behind non-obvious decisions, or summary for large diffs. Concise. Blank line separates from subject.

### Breaking changes

Add `!` after scope: `feat(api)!: change auth endpoint response format`
Include `BREAKING CHANGE:` footer explaining what changed.

## Step 4: Stage files and commit

Stage files, then commit. Always use HEREDOC format:

```
git add <files>
git commit -m "$(cat <<'EOF'
type(scope): description

Optional body explaining why, not what.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"
```

Rules:

- Always append `Co-Authored-By` trailer as last line
- Never use `--no-verify` or skip hooks
- If pre-commit hook fails, fix issue and create NEW commit (never amend)
- After committing, run `git status` to verify success

## Step 5: Multi-commit workflow

When splitting into multiple commits, work one at a time:

1. Stage files for first logical group
2. Commit w/ appropriate message
3. Move to next group
4. Repeat

Present plan to user before executing (e.g., "I'll make 3 commits: 1. feat(db): add user migrations, 2. feat(auth): add authentication service, 3. feat(handler): add auth HTTP endpoints"). Get confirmation, then proceed.

## Examples

```
feat(billing): add Stripe checkout session endpoint
```

```
fix(auth): prevent token reuse after refresh rotation
```

```
refactor(fw): extract OAuth provider interface

Decouple OAuth logic from auth service to support
pluggable providers (Google, GitHub, generic OIDC).

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

```
chore(deps): update go.mod dependencies
```

```
feat(db): add team and membership tables
```
