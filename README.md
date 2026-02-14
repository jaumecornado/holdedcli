# holdedcli

A Go CLI to connect to the Holded API.

## Commands

- `holded auth set --api-key <key>`
- `holded auth status`
- `holded ping`

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

## Examples

```bash
holded auth set --api-key "$HOLDED_API_KEY"
holded auth status
holded ping
holded ping --json
```

## macOS distribution

The project includes a GoReleaser release pipeline that generates:

- `darwin/amd64` and `darwin/arm64` binaries
- `checksums.txt`
- Homebrew tap publication

Expected GitHub Actions release secrets:

- `HOMEBREW_TAP_OWNER`
- `HOMEBREW_TAP_NAME`
- `HOMEBREW_TAP_GITHUB_TOKEN`
