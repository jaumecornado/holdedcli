# holdedcli

A Go CLI to connect to the Holded API.

## Installation

### Homebrew (recommended for macOS)

```bash
brew tap jaumecornado/tap
brew install holded
holded help
```

Example:

```bash
brew tap jaumecornado/tap
brew install holded
```

### Direct binary install (macOS)

```bash
# Apple Silicon
curl -L -o holded.tar.gz \
  https://github.com/jaumecornado/holdedcli/releases/latest/download/holdedcli_<version>_darwin_arm64.tar.gz

# Intel
curl -L -o holded.tar.gz \
  https://github.com/jaumecornado/holdedcli/releases/latest/download/holdedcli_<version>_darwin_amd64.tar.gz

tar -xzf holded.tar.gz
chmod +x holded
mv holded /usr/local/bin/holded
holded help
```

### Go install (development)

```bash
go install github.com/jaumecornado/holdedcli/cmd/holded@latest
holded help
```

## Commands

- `holded auth set --api-key <key>`
- `holded auth status`
- `holded ping`
- `holded actions list`
- `holded actions describe <action-id|operation-id>`
- `holded actions run <action-id|operation-id>`
- `holded actions run invoice.attach-file --path docType=purchase --path documentId=<id> --file ./ticket.jpg`

## Action Catalog (for skills)

For skill development and offline reference, this repository includes a versioned
snapshot of action IDs and endpoints:

- `docs/actions.md` (human-readable catalog)
- `docs/actions.json` (machine-readable catalog)

These files are generated from the official Holded API docs and can be updated
when needed. Runtime discovery remains available through:

- `holded actions list`

Global options:

- `--json` stable output for automations/skills.

Credential resolution order:

1. `--api-key`
2. `HOLDED_API_KEY`
3. `~/.config/holdedcli/config.yaml`

## Local build

```bash
go build -o holded ./cmd/holded
./holded help
```

## VS Code (development)

This repo includes ready-to-use VS Code debug profiles in
`.vscode/launch.json` to run the CLI directly from source (`cmd/holded`).

Setup:

```bash
cp .vscode/.env.example .vscode/.env
# edit .vscode/.env and set HOLDED_API_KEY
```

Then open Run and Debug in VS Code and use:

- `holded: actions list (dev)`
- `holded: actions run (prompt)`
- `holded: get contact by id (prompt)`

The debug config uses `HOLDED_CONFIG_PATH=${workspaceFolder}/.tmp/holdedcli/config.yaml` so local dev runs do not modify your global CLI config.

## Examples

```bash
holded auth set --api-key "$HOLDED_API_KEY"
holded auth status
holded ping
holded ping --json

# list all documented Holded actions
holded actions list

# filter actions
holded actions list --filter contacts

# inspect accepted parameters/body for one action
holded actions describe invoice.list-documents --json

# run with body validation (fails fast if unknown/missing/invalid top-level fields)
holded actions run invoice.create-contact --body '{"nam":"Acme"}' --json

# run an action by id
holded actions run invoice.list-contacts

# run an action by operation id with path/query params
holded actions run "Get Contact" --path contactId=abc123 --query customId=my-ref

# run an action with multipart file upload
holded actions run invoice.attach-file \
  --path docType=purchase \
  --path documentId=abc123 \
  --file ./ticket.jpg

# machine-readable output
holded actions run invoice.list-contacts --json
```

`holded actions` dynamically loads the current OpenAPI action catalog from
`https://developers.holded.com/reference/api-key`.

`holded actions run` validates `--body` against action metadata before sending
the request and returns `INVALID_BODY_PARAMS` when invalid.

## macOS distribution

The project includes a GoReleaser release pipeline that generates:

- `darwin/amd64` and `darwin/arm64` binaries
- `checksums.txt`
- Homebrew tap publication

Expected GitHub Actions release secrets:

- `HOMEBREW_TAP_OWNER=jaumecornado`
- `HOMEBREW_TAP_NAME=homebrew-tap`
- `HOMEBREW_TAP_GITHUB_TOKEN`
