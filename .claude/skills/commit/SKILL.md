| name | description | user_invocable |
| --- | --- | --- |
| commit | Create git commits using Conventional Commits format. Use this skill whenever the user asks to commit, make a commit, save changes, or says /commit. Also trigger when the user mentions 'conventional commit', 'commit message', or wants to stage and commit code changes. This skill handles analyzing diffs, auto-detecting the commit type and scope, generating well-formatted messages, and optionally splitting changes into multiple logical commits. | true |

## Conventional Commits Skill

Create professional git commits following the Conventional Commits specification. This skill analyzes staged and unstaged changes, determines the appropriate commit type and scope, generates a clear commit message, and executes the commit.

## Step 1: Gather context

Run these commands in parallel to understand the current state:

1. `git status` — see what's changed (never use `-uall` flag)
2. `git diff --staged` — see what's already staged
3. `git diff` — see unstaged changes
4. `git log --oneline -5` — see recent commit style for consistency

## Step 2: Analyze changes and decide on commit strategy

Look at all the changes holistically. Determine:

- __Are there multiple logical change groups?__ For example, a new feature in one package plus a bug fix in another, or migrations + service code + handler code that each represent a distinct unit of work. If so, ask the user whether they'd like to split into multiple commits. Suggest a split when changes touch unrelated areas — but don't over-split. Related changes (e.g., a service + its handler + its tests) belong in one commit.
- __What files should be staged?__ If nothing is staged yet, figure out which files belong together. Never stage files that look like secrets (`.env`, credentials, keys). Prefer `git add <specific files>` over `git add -A` or `git add .`.

## Step 3: Determine type, scope, and description

### Type

Pick the type based on the nature of the change, not just which files changed:

| Type | When to use |
| --- | --- |
| `feat` | A new feature or capability that didn't exist before |
| `fix` | A bug fix — something was broken and now it works |
| `refactor` | Code restructuring that doesn't change behavior |
| `docs` | Documentation only (README, comments, docstrings) |
| `chore` | Maintenance tasks: deps, configs, build scripts, tooling |
| `test` | Adding or updating tests only |
| `ci` | CI/CD pipeline changes |
| `perf` | Performance improvement |
| `style` | Code formatting, whitespace, semicolons (no logic change) |

If a commit includes a feature along with its tests, use `feat` — the type reflects the primary intent.

### Scope

Derive the scope from the primary area of the codebase affected. Use short, recognizable names:

- For Go: package name or directory (e.g., `auth`, `billing`, `middleware`, `fw`, `team`)
- For JS/TS: component or module name (e.g., `dashboard`, `api`, `hooks`)
- For config/infra: the system (e.g., `docker`, `ci`, `deps`)
- For migrations: `db` or `migrations`
- If changes span many areas with no clear primary, omit the scope

### Description

- Lowercase, imperative mood ("add", "fix", "remove" — not "added", "fixes", "removed")
- No period at the end
- Keep it under 72 characters
- Describe _what_ the commit does, not _how_

Good: `feat(auth): add JWT refresh token rotation`
Bad: `feat(auth): Added JWT refresh token rotation.`

### Body (optional)

Only include a body if the changes need explanation — the "why" behind a non-obvious decision, or a summary when the diff is large and complex. Keep it concise. Separate from the subject with a blank line.

### Breaking changes

If the commit introduces a breaking change, add `!` after the scope: `feat(api)!: change auth endpoint response format`
And include a `BREAKING CHANGE:` footer explaining what changed.

## Step 4: Stage files and commit

Stage the appropriate files, then create the commit. Always use HEREDOC format for the commit message to preserve formatting:

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

- Always append the `Co-Authored-By` trailer as the last line
- Never use `--no-verify` or skip hooks
- If a pre-commit hook fails, fix the issue and create a NEW commit (never amend)
- After committing, run `git status` to verify success

## Step 5: Multi-commit workflow

When splitting into multiple commits, work through them one at a time:

1. Stage files for the first logical group
2. Commit with the appropriate message
3. Move to the next group
4. Repeat until all changes are committed

Present the plan to the user before executing (e.g., "I'll make 3 commits: 1. feat(db): add user migrations, 2. feat(auth): add authentication service, 3. feat(handler): add auth HTTP endpoints"). Get confirmation, then proceed.

## Examples

Single feature with tests:

```
feat(billing): add Stripe checkout session endpoint
```

Bug fix:

```
fix(auth): prevent token reuse after refresh rotation
```

Multiple areas, with body:

```
refactor(fw): extract OAuth provider interface

Decouple OAuth logic from auth service to support
pluggable providers (Google, GitHub, generic OIDC).

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

Chore:

```
chore(deps): update go.mod dependencies
```

Migration + schema:

```
feat(db): add team and membership tables
```
