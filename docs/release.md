# Release Checklist

This project ships release assets from GitHub Releases only (no APT repository/channel).

## 1) Prepare and tag a release

1. Update version references and changelog entries.
2. Commit release prep changes.
3. Create and push a tag:

```bash
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin vX.Y.Z
```

## 2) Run the release workflow

1. Confirm the `Release` GitHub Actions workflow starts from the `v*` tag push.
2. The workflow runs:
   - `go test ./...`
   - GoReleaser preflight build (`--skip=publish`)
   - `.deb` artifact verification
   - GoReleaser publish

## 3) Verify release assets

For tag `vX.Y.Z`, confirm GitHub Release assets include all of the following:

- `awt_X.Y.Z_Linux_x86_64.tar.gz`
- `awt_X.Y.Z_Linux_arm64.tar.gz`
- `awt_X.Y.Z_Darwin_x86_64.tar.gz`
- `awt_X.Y.Z_Darwin_arm64.tar.gz`
- `checksums.txt`
- `awt_X.Y.Z_linux_amd64.deb`
- `awt_X.Y.Z_linux_arm64.deb`

## 4) Post-release install tests

1. Installer script test (Linux/macOS):

```bash
curl -fsSL https://raw.githubusercontent.com/kernel-labs-ai/awt/main/scripts/install.sh | bash
awt version
```

2. Debian package test (Ubuntu example):

```bash
sudo apt install ./awt_X.Y.Z_linux_amd64.deb
awt version
```

If `apt install` is not available for local file install in your environment:

```bash
sudo dpkg -i ./awt_X.Y.Z_linux_amd64.deb
awt version
```
