# gitcrn-cli

Private CLI helper for a fixed Gitea SSH host (`gitcrn`).

End users do not need Go. They install prebuilt binaries from Gitea Releases.

## Commands

```bash
gitcrn init --default
gitcrn init --custom --host 100.91.132.35 --port 222 --user git

gitcrn clone vltc/kapri
gitcrn remote add gitcrn vltc/crnbg

gitcrn -v
gitcrn --version
```

## What `init --default` writes

`~/.ssh/config` (Linux/macOS) or `%USERPROFILE%\.ssh\config` (Windows):

```sshconfig
Host gitcrn
    HostName 100.91.132.35
    User git
    Port 222
```

## Install

Linux (download binary from release):

```bash
./scripts/install.sh --version latest
```

If repo is private, pass token (recommended via env var):

```bash
GITEA_TOKEN="<your-token>" ./scripts/install.sh --version latest
```

Or:

```bash
./scripts/install.sh --version latest --token "<your-token>"
```

If your Gitea URL/owner/repo differs:

```bash
./scripts/install.sh \
  --server-url "http://100.91.132.35:5000" \
  --owner "vltc" \
  --repo "gitcrn-cli" \
  --version latest
```

If your cert is self-signed:

```bash
./scripts/install.sh --version latest --insecure
```

Linux with custom install path:

```bash
./scripts/install.sh --prefix "$HOME/.local/bin" --version latest
```

Windows (PowerShell, download binary from release):

```powershell
.\scripts\install.ps1 -Version latest
```

If repo is private, pass token:

```powershell
$env:GITEA_TOKEN = "<your-token>"
.\scripts\install.ps1 -Version latest
```

Or:

```powershell
.\scripts\install.ps1 -Version latest -Token "<your-token>"
```

If your Gitea URL/owner/repo differs:

```powershell
.\scripts\install.ps1 `
  -ServerUrl "http://100.91.132.35:5000" `
  -Owner "vltc" `
  -Repo "gitcrn-cli" `
  -Version latest
```

If your cert is self-signed:

```powershell
.\scripts\install.ps1 -Version latest -Insecure
```

## Update

Linux:

```bash
./scripts/update.sh
```

Windows (PowerShell):

```powershell
.\scripts\update.ps1
```

## Maintainer: build release assets

For publishing a new version (Go required on maintainer machine):

```bash
./scripts/release.sh --version v0.1.0
```

It creates files in `dist/`:

```text
gitcrn-linux-amd64
gitcrn-linux-arm64
gitcrn-windows-amd64.exe
gitcrn-windows-arm64.exe
checksums.txt
```

Upload those files to the matching Gitea release tag (`v0.1.0`).

## Dev (source build)

```bash
make test
make build
make build-linux
make build-windows
make release-assets VERSION=v0.1.0
```
