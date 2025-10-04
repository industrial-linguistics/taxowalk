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
- `--history-db` – SQLite database path to track token usage history (optional).
- `--debug` – write verbose diagnostic logging to stderr.
- `--refresh-taxonomy` – bypass the cached taxonomy and fetch a fresh copy.
- `--version` – print the installed taxowalk version and exit.

The command prints the selected taxonomy `full_name` followed by its canonical ID.

### Examples

```bash
taxowalk "Handmade leather tote bag"
cat product.txt | taxowalk --stdin
taxowalk --history-db usage.db "Wireless headphones"
```

## Token Usage Tracking

When you provide the `--history-db` flag, taxowalk records each classification along with token usage to a SQLite database. Use `taxowalk-report` to analyze this data.

### taxowalk-report

```bash
taxowalk-report [flags]
```

#### Flags

- `--db` – SQLite database path (required).
- `--all` – show all classification records with details.
- `--check-24h` – check if token usage in the last 24 hours exceeds the limit.
- `--limit` – token limit for 24-hour check (default: 5000000).

#### Examples

```bash
# Show summary
taxowalk-report --db usage.db

# Show all records
taxowalk-report --db usage.db --all

# Check if you've exceeded 5M tokens in the last 24 hours
taxowalk-report --db usage.db --check-24h

# Check with custom limit
taxowalk-report --db usage.db --check-24h --limit 1000000
```

The `--check-24h` flag exits with code 2 if the limit is exceeded, making it suitable for automation.

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

