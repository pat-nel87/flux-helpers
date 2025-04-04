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
```

### 🐳 Using flux-helpers with Docker
🚀 Run without installing Go
You can run flux-helpers fully containerized, no local Go install required:

```bash
docker run --rm \
  -v $PWD:/workdir \
  ghcr.io/your-org/flux-helpers:latest \
  bump \
  --file /workdir/test_files/multiple-bump.yaml \
  --set ghcr.io/my-org/my-api=1.3.99 \
  --dry-run
```
### 🧪 Run Unit Tests in Docker

```bash
docker build --build-arg TEST=true -t flux-helpers:test .
```

### 🧬 Run Fuzz Testing 

```bash
docker build \
  --build-arg FUZZ=true \
  --build-arg FUZZTIME=1m \
  -t flux-helpers:fuzz .
```
### 🧰 Helm Helpers
inject-helm-condition
Automatically injects a conditional imagePullSecrets block into a Helm chart’s deployment.yaml and updates values.yaml accordingly.

```bash
flux-helpers inject-helm-condition --chart ./charts/my-service
```

What it does:
🔧 Finds any templates/*deployment.yaml in your chart directory

🩺 Injects the following block under spec.template.spec:

```yaml
{{- if .Values.image.imagePullSecret }}
imagePullSecrets:
  - name: {{ .Values.image.imagePullSecret }}
{{- end }}
```

📝 Ensures values.yaml contains:

```yaml
image:
  imagePullSecret: ""
```

### 🔍 Renders the modified chart using Helm libraries for preview/debug

Example with Docker:
```bash
docker run --rm \
  -v $PWD:/chart \
  ghcr.io/your-org/flux-helpers:latest \
  inject-helm-condition --chart /chart
This helper is ideal for automating image pull secret logic across multiple charts in your GitOps pipeline.
```

