# deploy-tool

`deploy-tool` is a Go CLI that turns a simple deployment manifest into the GitLab CI jobs needed to push Helm charts to multiple clusters. It reads a YAML config describing each deployment target and produces a ready-to-commit pipeline fragment.

## Features
- Validates deployment definitions and prevents duplicate job names.
- Builds multi-stage GitLab pipelines with optional promotion gates between environments.
- Logs the generated pipeline so you can redirect it to a file or inspect it in the console.

## Getting Started
### Prerequisites
- Go 1.24 or newer.

### Clone and build
```bash
git clone <repo-url>
cd deploy-tool
go build ./...
```

### Run the CLI
Use the `generate-app-pipeline` subcommand and point it at a deployment config:

```bash
go run . generate-app-pipeline --config-file deploycicd/app-config-file.yaml
```

The command logs the resulting pipeline definition. Redirect the output if you want to capture it:

```bash
go run . generate-app-pipeline --config-file my-config.yaml 2>&1 | tee pipeline.gitlab-ci.yml
```

## Configuration file
Provide a YAML file matching the schema below. See `deploycicd/app-config-file.yaml` for a complete example.

```yaml
deployments:
  - instanceName: "my-app"
    valueFile: "value-dev.yaml"
    targetCluster: "dev-01"
    env: "dev"
    pathToProd: true
```

- `instanceName` (string) – logical application release name.
- `valueFile` (string) – Helm values file to deploy.
- `targetCluster` (string) – GitLab environment/cluster target.
- `env` (string) – deployment stage (`dev`, `preprod`, or `prod`).
- `pathToProd` (bool, optional) – flag promotions should follow this deployment when building `needs` dependencies.

When `pathToProd` is true on `dev` and `preprod` entries, the generator wires later environments to depend on the previous stage, ensuring manual promotion flow.

### Generated pipeline outline
A pipeline generated from the sample config resembles:

```yaml
include:
  - project: "template"
    ref: "master"
    file:
      - "gitlab-ci/templates/tmplate.gitlab-ci.yaml"

push-my-app-to-dev-01:
  variables:
    VALUE_FILE: value-dev.yaml
    TARGET_CLUSTER: dev-01
    INSTANCENAME: my-app
    ENV: dev
  stage: Push Manifests Dev
  extends: .push-to-target-cluster-repo
  when: manual
```

Additional jobs are generated for every deployment entry; `preprod` and `prod` stages gain `needs` dependencies when a matching `pathToProd` chain exists.

## Development
- Run `go fmt ./...` to keep formatting consistent.
- Execute `go run . --help` to see Cobra-generated usage information.
- Add automated tests in `deploycicd` as the pipeline generation logic grows.

## License
This project currently carries the default Cobra license header; update it if you adopt a different license.
