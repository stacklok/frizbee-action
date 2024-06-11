![image](https://github.com/stacklok/frizbee/assets/16540482/35034046-d962-475d-b8e2-67b7625f2a60)

---
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache2.0-brightgreen.svg)](https://opensource.org/licenses/Apache-2.0) | [![](https://dcbadge.vercel.app/api/server/RkzVuTp3WK?logo=discord&label=Discord&color=5865&style=flat)](https://discord.gg/RkzVuTp3WK)

---
# Frizbee Action

Frizbee Action helps you pin your GitHub Actions and container images to specific versions using checksums.

You can configure it to fix it all for you and open a PR with the proposed changes,
fail the CI if unpinned actions are found and much more. 

The action is based on the Frizbee tool, available both as a CLI and as a library - https://github.com/stacklok/frizbee

## Table of Contents

- [Usage](#usage)
- [Configuration](#configuration)
- [Contributing](#contributing)
- [License](#license)

## Usage

To use the Frizbee Action, you can use the following methods:

```bash
name: Frizbee Pinned Actions and Container Images Check

on:
  schedule:
    - cron: '0 0 * * *' # Run every day at midnight
  workflow_dispatch:

jobs:
  frizbee_check:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
      - uses: stacklok/frizbee-action@v0.0.1
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        with:
          actions: .github/workflows
          dockerfiles: ./docker
          kubernetes: ./k8s
          docker_compose: ./docker
          open_pr: true
          fail_on_unpinned: true
```

## Configuration

The Frizbee Action can be configured through the following inputs:

```yml
  actions:
    description: "Actions to correct"
    required: false
    default: ".github/workflows"
  dockerfiles:
    description: "Dockerfiles to correct"
    required: false
    default: "Dockerfile"
  kubernetes:
    description: "Kubernetes manifests to correct"
    required: false
    default: ""
  docker_compose:
    description: "Docker Compose files to correct"
    required: false
    default: ""
  open_pr:
    description: "Open a PR with the changes"
    required: false
    default: "true"
  fail_on_unpinned:
    description: "Fail if an unpinned action/image is found"
    required: false
    default: "false"
```

### Limitations

The default `GITHUB_TOKEN` doesn't have the necessary permissions (`workflows`) to open a PR. 
In case you want to use the `open_pr` feature, you will need to create a new token with the correct scope, add it as a secret
and pass it to the action through the `GITHUB_TOKEN` environment variable.

## Contributing

We welcome contributions to Frizbee Action. Please see our [Contributing](./CONTRIBUTING.md) guide for more information.

## License

Frizbee is licensed under the [Apache 2.0 License](./LICENSE).