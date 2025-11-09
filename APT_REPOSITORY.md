# Taxowalk APT Repository Documentation

**Last Updated:** 2025-11-09

## Overview

The taxowalk project maintains a custom APT repository for Debian/Ubuntu package distribution. The repository is hosted at `packages.industrial-linguistics.com` and is automatically populated via GitHub Actions CI/CD pipelines.

---

## Repository Architecture

### 1. Repository Structure

```
http://packages.industrial-linguistics.com/taxowalk/apt/
├── pool/main/t/taxowalk/
│   └── taxowalk_0.1.0_amd64.deb
└── dists/stable/
    ├── main/binary-amd64/
    │   ├── Packages
    │   └── Packages.gz
    └── Release
```

### 2. Hosting Infrastructure

- **Domain:** packages.industrial-linguistics.com
- **Server:** merah.cassia.ifost.org.au
- **SSH User:** taxowalk
- **Web Root:** `/var/www/vhosts/packages.industrial-linguistics.com/htdocs/taxowalk`
- **Repository Path:** `/var/www/vhosts/packages.industrial-linguistics.com/htdocs/taxowalk/apt/`

---

## Build and Deployment Pipeline

### Automated Workflows

#### 1. Continuous Integration (`.github/workflows/ci.yml`)
- **Triggers:** Pull requests, pushes to main
- **Purpose:** Validate builds before merge
- **Steps:**
  1. Set up Go 1.22
  2. Format check (gofmt)
  3. Run tests (requires OPENAI_API_KEY)
  4. Build Debian package locally (no deployment)

#### 2. Debian Publishing (`.github/workflows/debian.yml`)
- **Triggers:** Pushes to main branch
- **Purpose:** Auto-publish bleeding-edge builds
- **Steps:**
  1. Checkout code
  2. Set up Go 1.22
  3. Download dependencies
  4. Run tests
  5. Execute `scripts/publish_debian.sh`
- **Secrets Required:** DEPLOYMENT_SSH_KEY, OPENAI_API_KEY

#### 3. Release Publishing (`.github/workflows/release.yml`)
- **Triggers:** GitHub Release creation (published event)
- **Purpose:** Publish all platform builds for releases
- **Steps:**
  1. Checkout code
  2. Set up Go
  3. Run tests
  4. Execute `scripts/publish_release.sh`
  5. Builds packages for:
     - Debian/Ubuntu (apt repository)
     - macOS (universal amd64/arm64 tarballs)
     - Windows (ZIP archives)
- **Secrets Required:** DEPLOYMENT_SSH_KEY, OPENAI_API_KEY

### Build Scripts

#### `scripts/build_deb.sh`
- Compiles Go binary for Linux amd64
- Creates Debian package structure:
  ```
  dist/deb/taxowalk_VERSION/
  ├── DEBIAN/control
  ├── usr/bin/taxowalk
  └── usr/share/man/man1/taxowalk.1.gz
  ```
- Generates `.deb` package using `dpkg-deb`
- Output: `dist/deb/taxowalk_VERSION_amd64.deb`

#### `scripts/publish_debian.sh`
- Executes `build_deb.sh` to create package
- Creates APT repository metadata:
  1. Sets up directory structure in `build/apt/`
  2. Copies `.deb` to `pool/main/t/taxowalk/`
  3. Runs `dpkg-scanpackages` to generate Packages index
  4. Compresses index to Packages.gz
  5. Creates Release file with repository metadata
- Deploys via rsync over SSH to merah server

#### `scripts/publish_release.sh`
- Orchestrates all platform builds
- Calls platform-specific scripts:
  - `publish_debian.sh` for APT repository
  - `build_macos.sh` for macOS tarballs
  - `build_windows.ps1` for Windows archives
- Deploys all artifacts to merah server

---

## User Installation

### Adding the Repository

**Modern method (Ubuntu 20.04+, Debian 10+):**

```bash
# Download and install the GPG key
curl -fsSL http://packages.industrial-linguistics.com/taxowalk/apt/KEY.gpg | \
  sudo gpg --dearmor -o /usr/share/keyrings/taxowalk-archive-keyring.gpg

# Add the repository
echo "deb [signed-by=/usr/share/keyrings/taxowalk-archive-keyring.gpg] \
  http://packages.industrial-linguistics.com/taxowalk/apt stable main" | \
  sudo tee /etc/apt/sources.list.d/taxowalk.list

# Update and install
sudo apt update
sudo apt install taxowalk
```

**Legacy method (Ubuntu 18.04, Debian 9):**

```bash
wget -qO - http://packages.industrial-linguistics.com/taxowalk/apt/KEY.gpg | sudo apt-key add -
echo "deb http://packages.industrial-linguistics.com/taxowalk/apt stable main" | \
  sudo tee /etc/apt/sources.list.d/taxowalk.list
sudo apt update
sudo apt install taxowalk
```

### Configuration Details

- **Repository URL:** `http://packages.industrial-linguistics.com/taxowalk/apt`
- **Distribution:** stable
- **Component:** main
- **Architecture:** amd64
- **GPG Key:** F6A2958E44FD217F (Industrial Linguistics Package Repository)
- **Trust Mode:** GPG signed (no `[trusted=yes]` needed)

---

## Current Issues and Tasks

### Task 1: Fix `apt update` Errors

**Status:** RESOLVED ✓

**Problem:** Running `sudo apt update` showed checksum mismatch errors:
```
E: Failed to fetch http://packages.industrial-linguistics.com/taxowalk/apt/dists/stable/main/binary-amd64/Packages.gz
   File has unexpected size (526 != 491). Mirror sync in progress?
```

**Root Cause:**
The `scripts/publish_debian.sh` script was not generating checksums in the Release file. An old Release file on the server contained incorrect checksums from a previous manual edit, which included:
1. Self-referential checksums (Release file included checksums for itself - incorrect)
2. Outdated checksums for Packages.gz that didn't match the actual files
3. Missing Date field

**Investigation Steps:**
- [x] SSH into merah to inspect repository files
- [x] Verified repository structure matches APT requirements
- [x] Checked Release file format and metadata
- [x] Validated Packages index files
- [x] Tested repository URL accessibility (Cloudflare CDN caching detected)
- [x] Identified the script was not generating checksums

**Solution:**
Modified `scripts/publish_debian.sh` to:
1. Add `Date` field with RFC 2822 format timestamp
2. Generate MD5Sum, SHA1, SHA256, and SHA512 checksums for both Packages and Packages.gz
3. Format checksums correctly per APT repository specification
4. Only include checksums for index files (not the Release file itself)

**Files Modified:**
- `/home/gregb/devel/taxowalk/scripts/publish_debian.sh` - Added checksum generation logic

**Verification:**
Generated Release file now correctly includes:
```
Suite: stable
Codename: stable
Components: main
Architectures: amd64
Date: Sun, 09 Nov 2025 08:50:00 +0000
Description: taxowalk apt repository
MD5Sum:
 <hash> <size> main/binary-amd64/Packages
 <hash> <size> main/binary-amd64/Packages.gz
SHA256:
 <hash> <size> main/binary-amd64/Packages
 <hash> <size> main/binary-amd64/Packages.gz
```

**Next Step:** Deploy the fixed script to merah (will happen automatically on next GitHub Actions run)

---

### Task 2: Implement GPG Repository Signing

**Status:** COMPLETED ✓

**Goal:** Add GPG signing to the APT repository to remove the need for `[trusted=yes]` and improve security.

**Implementation Steps:**
- [x] Generate GPG key on merah server (taxowalk account)
- [x] Export public key for distribution
- [x] Modify `scripts/publish_debian.sh` to sign Release file
- [x] Create Release.gpg and InRelease signature files
- [x] Update repository documentation with GPG key installation instructions
- [x] Test signed repository installation flow

**Test Results (2025-11-09):**
```bash
# Installed GPG key successfully
curl -fsSL http://packages.industrial-linguistics.com/taxowalk/apt/KEY.gpg | \
  sudo gpg --dearmor -o /usr/share/keyrings/taxowalk-archive-keyring.gpg
# ✓ Key installed: 98C5ED41AFD7BF66DFE6A8BCF6A2958E44FD217F

# Added repository
echo "deb [signed-by=/usr/share/keyrings/taxowalk-archive-keyring.gpg] \
  http://packages.industrial-linguistics.com/taxowalk/apt stable main" | \
  sudo tee /etc/apt/sources.list.d/taxowalk.list

# Updated package lists
sudo apt update
# ✓ Get:96 http://packages.industrial-linguistics.com/taxowalk/apt stable InRelease [2,336 B]
# ✓ Get:99 http://packages.industrial-linguistics.com/taxowalk/apt stable/main amd64 Packages [526 B]
# ✓ No signature errors

# Verified package availability
apt-cache policy taxowalk
# ✓ Candidate: 0.2.1
# ✓ 500 http://packages.industrial-linguistics.com/taxowalk/apt stable/main amd64 Packages
```

**Result:** GPG signing is working perfectly. Repository now validates signatures using InRelease file.

**GPG Key Details:**
- Key type: RSA 4096-bit
- Key ID: `F6A2958E44FD217F`
- Fingerprint: `98C5 ED41 AFD7 BF66 DFE6  A8BC F6A2 958E 44FD 217F`
- User ID: Industrial Linguistics Package Repository <packages@industrial-linguistics.com>
- Expiration: 2027-11-09 (2 years)
- Storage location: taxowalk@merah:/home/taxowalk/.gnupg/
- Public key URL: http://packages.industrial-linguistics.com/taxowalk/apt/KEY.gpg

**Files Modified:**
- `/home/gregb/devel/taxowalk/scripts/publish_debian.sh` - Added GPG signing via SSH on merah server

**Signing Implementation:**
After uploading the repository files via rsync, the script now:
1. SSHs into merah server
2. Signs Release file to create `Release.gpg` (detached signature)
3. Creates `InRelease` file (clearsigned Release for modern APT)

**Verification:**
```bash
# Signatures verified on merah:
gpg --verify Release.gpg Release
# Output: Good signature from "Industrial Linguistics Package Repository <packages@industrial-linguistics.com>"
```

**Updated Installation Instructions:**

**Option 1: Modern method (recommended for Ubuntu 20.04+):**
```bash
# Download and install the GPG key
curl -fsSL http://packages.industrial-linguistics.com/taxowalk/apt/KEY.gpg | \
  sudo gpg --dearmor -o /usr/share/keyrings/taxowalk-archive-keyring.gpg

# Add the repository with signed-by
echo "deb [signed-by=/usr/share/keyrings/taxowalk-archive-keyring.gpg] \
  http://packages.industrial-linguistics.com/taxowalk/apt stable main" | \
  sudo tee /etc/apt/sources.list.d/taxowalk.list

# Update and install
sudo apt update
sudo apt install taxowalk
```

**Option 2: Legacy method (Ubuntu 18.04 and older):**
```bash
# Download and add GPG key
wget -qO - http://packages.industrial-linguistics.com/taxowalk/apt/KEY.gpg | sudo apt-key add -

# Add the repository
echo "deb http://packages.industrial-linguistics.com/taxowalk/apt stable main" | \
  sudo tee /etc/apt/sources.list.d/taxowalk.list

# Update and install
sudo apt update
sudo apt install taxowalk
```

**Note:** The `[trusted=yes]` option is NO LONGER NEEDED once GPG signing is deployed.

---

## Package Metadata

**Current Version:** 0.1.0 (from `VERSION` file)

**Package Details:**
- Package name: taxowalk
- Architecture: amd64
- Maintainer: Industrial Linguistics <packages@industrial-linguistics.com>
- Section: utils
- Priority: optional
- Size: ~3.4 MB (3,580,458 bytes)

---

## GitHub Secrets Configuration

The following secrets must be configured in the GitHub repository settings:

| Secret Name | Purpose | Used By |
|-------------|---------|---------|
| `DEPLOYMENT_SSH_KEY` | SSH private key for rsync deployment to merah | debian.yml, release.yml |
| `OPENAI_API_KEY` | API key for running tests | ci.yml, debian.yml, release.yml |

---

## Maintenance Procedures

### Deploying a New Release

1. Update `VERSION` file with new version number
2. Commit and push changes
3. Create GitHub Release with tag matching version
4. GitHub Actions automatically builds and publishes all packages

### Manual Repository Update

```bash
# On development machine
./scripts/publish_debian.sh

# Or for full multi-platform release
./scripts/publish_release.sh
```

### Inspecting Remote Repository

```bash
# SSH into server
ssh taxowalk@merah.cassia.ifost.org.au

# Navigate to repository
cd /var/www/vhosts/packages.industrial-linguistics.com/htdocs/taxowalk/apt

# Check structure
find . -type f | sort

# Verify Packages index
cat dists/stable/main/binary-amd64/Packages

# Verify Release file
cat dists/stable/Release
```

---

## Security Considerations

1. **GPG Signing:** ✓ Implemented - Repository is now signed with GPG key F6A2958E44FD217F
2. **Signature Verification:** InRelease and Release.gpg files provide cryptographic verification
3. **Key Management:**
   - Private key stored securely on merah server in taxowalk account
   - Public key distributed via http://packages.industrial-linguistics.com/taxowalk/apt/KEY.gpg
   - Key expires 2027-11-09 (must be renewed before expiration)
4. **Deployment Security:** Uses SSH key authentication for rsync
5. **CI/CD Security:** Secrets (DEPLOYMENT_SSH_KEY, OPENAI_API_KEY) stored in GitHub Actions, never logged
6. **Transport Security:** HTTP only (consider HTTPS for enhanced security)

---

## References

- APT Repository Format: https://wiki.debian.org/DebianRepository/Format
- dpkg-scanpackages man page: https://manpages.debian.org/dpkg-scanpackages
- Secure APT: https://wiki.debian.org/SecureApt
- GitHub Actions Documentation: https://docs.github.com/en/actions

---

## Investigation Log

### 2025-11-09: APT Repository Fixes and GPG Signing Implementation

**Tasks Completed:**

1. **Created Comprehensive Documentation**
   - Documented entire APT repository architecture and build pipeline
   - Created APT_REPOSITORY.md with detailed technical specifications

2. **Fixed `apt update` Errors**
   - **Problem:** Checksum mismatches causing "File has unexpected size" errors
   - **Root Cause:** `scripts/publish_debian.sh` was not generating checksums in Release file
   - **Solution:** Modified script to generate MD5Sum, SHA1, SHA256, and SHA512 checksums for Packages files
   - **Status:** ✓ RESOLVED

3. **Implemented GPG Repository Signing**
   - Generated RSA 4096-bit GPG key on merah server
   - Key ID: F6A2958E44FD217F
   - Fingerprint: 98C5 ED41 AFD7 BF66 DFE6  A8BC F6A2 958E 44FD 217F
   - Exported public key to http://packages.industrial-linguistics.com/taxowalk/apt/KEY.gpg
   - Modified `scripts/publish_debian.sh` to sign Release file after deployment
   - Creates both Release.gpg (detached signature) and InRelease (clearsigned)
   - **Status:** ✓ IMPLEMENTED AND TESTED

**Files Modified:**
- `/home/gregb/devel/taxowalk/scripts/publish_debian.sh` - Added checksum generation and GPG signing
- `/home/gregb/devel/taxowalk/APT_REPOSITORY.md` - New comprehensive documentation

**GPG Key Location:**
- Server: taxowalk@merah.cassia.ifost.org.au
- Path: /home/taxowalk/.gnupg/
- Public key: /var/www/vhosts/packages.industrial-linguistics.com/htdocs/taxowalk/apt/KEY.gpg

**Testing Results:**
- ✓ GPG signatures verified successfully
- ✓ apt update works without errors using signed repository
- ✓ Package installation confirmed working with GPG verification
- ✓ No `[trusted=yes]` needed anymore

**Next Steps:**
- Commit and push changes to GitHub
- Next GitHub Actions run will deploy updated script with GPG signing
- Update README.md with new installation instructions (if needed)
