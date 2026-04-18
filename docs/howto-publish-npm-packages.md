# How to publish npm packages

LiveBoard ships three public npm packages under the `@wriven` scope:

| Package | Source |
|---|---|
| `@wriven/liveboard-shared` | `web/shared/` |
| `@wriven/liveboard-shell` | `web/shell/` |
| `@wriven/liveboard-renderer-default` | `web/renderer/default/` |

## Prerequisites

1. An npm account with access to the `@wriven` org.
2. An **Automation** token from npmjs.com (Account → Access Tokens → Generate → Automation). Automation tokens bypass 2FA and are required for non-interactive publishing.
3. Token written to `~/.npmrc`:
   ```
   //registry.npmjs.org/:_authToken=npm_YOUR_TOKEN_HERE
   ```

## Steps

### 1. Bump versions

Update `"version"` in all three `package.json` files to match the release tag (e.g. `0.20.0`):

- `web/shared/package.json`
- `web/shell/package.json`
- `web/renderer/default/package.json`

### 2. Build and publish

```bash
make npm-publish
```

This runs `bun run build` / `bun run build:npm` for each package then `bun publish`.

### 3. Verify

```bash
npm info @wriven/liveboard-shared version
npm info @wriven/liveboard-shell version
npm info @wriven/liveboard-renderer-default version
```

## Notes

- The first time a package is published the `@wriven` npm org must already exist. Create it at npmjs.com → Organizations → Create if needed.
- Tokens expire — generate a new Automation token before publishing if the old one has expired or was revoked.
- Never commit tokens to the repo. Keep them in `~/.npmrc` only.
