# taxowalk

taxowalk classifies free-form product descriptions into the [Shopify product taxonomy](https://github.com/Shopify/product-taxonomy). It incrementally traverses the taxonomy by prompting OpenAI's `gpt-5-mini` model with the candidate categories that exist at each level until a leaf node (or "none of these") is selected.

## Features

- Command-line interface with stdin/CLI input modes.
- Automatic retrieval of the Shopify taxonomy JSON.
- Iterative prompting strategy that mirrors human browsing of the taxonomy.
- Default OpenAI API key discovery from `~/.openai.key` with CLI override.
- Linux man page and Windows help page.
- Debian package build pipeline for pull requests.
- Release automation that builds Linux, macOS, and Windows packages and publishes them to an apt repository served from `packages.industrial-linguistics.com/shopify` via SSH.

## Installation

### From source

```bash
go build ./cmd/taxowalk
```

### Debian/Ubuntu package

Pull requests automatically build a `.deb` using `scripts/build_deb.sh`. Release builds publish to an apt repository hosted at `http://packages.industrial-linguistics.com/shopify/`. After a release is deployed, install it with:

```bash
echo "deb [trusted=yes] http://packages.industrial-linguistics.com/shopify/apt stable main" | sudo tee /etc/apt/sources.list.d/taxowalk.list
sudo apt update
sudo apt install taxowalk
```

### macOS and Windows builds

Release workflows run the platform-specific build scripts:

- `scripts/build_macos.sh` produces universal tarballs containing the binary and man page.
- `scripts/build_windows.ps1` emits a ZIP with the Windows executable and help page.

The resulting archives are uploaded by the workflow to the deployment host via `rsync` using `DEPLOYMENT_SSH_KEY`.

## Usage

```bash
taxowalk [flags] [product description]
```

### Flags

- `--stdin` – read the description from standard input.
- `--openai-key` – override the OpenAI API key (otherwise uses `OPENAI_API_KEY` or `~/.openai.key`).
- `--openai-base-url` – point to a different OpenAI-compatible endpoint.
- `--taxonomy-url` – provide an alternate taxonomy JSON URL or file path.

The command prints the selected taxonomy `full_name` followed by its canonical ID.

### Examples

```bash
taxowalk "Handmade leather tote bag"
cat product.txt | taxowalk --stdin
```

## Development

- `go test ./...` runs the unit test suite. Integration tests that require OpenAI are skipped unless `OPENAI_API_KEY` is set.
- `scripts/build_deb.sh` builds a Debian package in `dist/deb/` without uploading it.
- `scripts/build_macos.sh` and `scripts/build_windows.ps1` cross-compile for the respective platforms; these scripts are invoked only by release workflows.
- `scripts/publish_release.sh` (used by the release workflow) builds all packages, assembles the apt repository metadata, and deploys the artifacts to `taxowalk@merah.cassia.ifost.org.au` via `rsync` using the `DEPLOYMENT_SSH_KEY` secret.

The `VERSION` file **must** be updated whenever a change may alter the executable or any installation package. Continuous integration checks ensure packages build before merging.

## Documentation

- Linux/macOS man page: `docs/taxowalk.1`
- Windows help page: `docs/taxowalk-help.txt`

## Security

The application reads the OpenAI API key from:

1. The `--openai-key` flag (highest precedence)
2. The `OPENAI_API_KEY` environment variable
3. `~/.openai.key` (one-line file)

GitHub Actions use repository secrets (`OPENAI_API_KEY`, `DEPLOYMENT_SSH_KEY`) to run tests and deploy release artifacts without persisting them to GitHub.

