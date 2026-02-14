# holdedcli

A Go CLI to connect to the Holded API.

## Installation

### Homebrew (recommended for macOS)

```bash
brew tap jaumecornado/homebrew-tap
brew install holded
holded help
```

Example:

```bash
brew tap jaumecornado/homebrew-tap
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

- `HOMEBREW_TAP_OWNER=jaumecornado`
- `HOMEBREW_TAP_NAME=homebrew-tap`
- `HOMEBREW_TAP_GITHUB_TOKEN`
