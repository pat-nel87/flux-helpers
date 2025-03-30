# flux-helpers

`flux-helpers` is a CLI tool designed to simplify the management of Flux GitOps manifests, particularly HelmRelease YAML files. It provides functionality for safely and automatically updating container image tags within these manifests.

## Features

- **Bump Image Tags**: Update one or more image tags in a HelmRelease YAML file.
- **Dry-Run Mode**: Preview changes without modifying the file.
- **Semantic Version Validation**: Ensures image tags conform to semantic versioning.
- **Nested Image Block Support**: Handles deeply nested image configurations.

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/pat-nel87/flux-helpers.git
   cd flux-helpers

2. Build the binary:
   ```bash
   go build -o flux-helpers

3. Add the binary to your PATH for easy access:
   ```bash
   mv flux-helpers /usr/local/bin/

## Usage

### 🧩 How It Works

### 🛠 Main Commands

**bump**
Update image versions in a HelmRelease YAML file.

```bash
flux-helpers bump \
  --file test_files/multiple-bump.yaml \
  --set ghcr.io/my-org/my-api=1.3.9 \
  --set envoyproxy/envoy=1.26.3 \
  --dry-run
Flags:
Flag	Description
--file, -f	Path to your HelmRelease YAML file
--set	One or more repository=version updates
--dry-run	If true, prints updates without writing file