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

```yml
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
      - uses: stacklok/frizbee-action@v0.0.2
        env:
          GITHUB_TOKEN: ${{ secrets.FRIZBEE_TOKEN }}
        with:
          actions: .github/workflows
          dockerfiles: ["./Dockerfile", "./images"] # You can specify multiple files or directories
          kubernetes: ["./k8s"]
          docker_compose: ["./docker"]
          open_pr: true
          fail_on_unpinned: true
```

### Fine-tuning the action

There are several options available to further exclude certain branches, images or actions from the check.

#### Exclude actions
The `actions_exclude` input allows you to exclude certain actions from the check. This is useful if you have actions that you don't want to pin.

```yml
with:
  actions_exclude: ["slsa-framework/slsa-github-generator/.github/workflows/generator_generic_slsa3.yml"]
```

Default: Unset. All actions are checked.

#### Exclude action branches
The `actions_exclude_branches` input allows you to exclude certain branches from the check. The reasoning being if you refer to an action by a branch in your workflow, you want to follow that branch.

```yml
with:
  actions_exclude: ["main"]
```
Default: Set to `*` meaning that actions that are referred to by a branch are never pinned.

#### Exclude container images
The `images_exclude` input allows you to exclude certain container images from the check. This is useful if you have images that you don't want to pin.

```yml
with:
  images_exclude: ["nginx"]
```

Default: `["scratch"]`

#### Exclude container image tags
The `images_exclude_tags` input allows you to exclude certain tags from the check. Some tags are not meant to be pinned, like `latest`.

```yml
with:
  images_exclude_tags: ["latest"]
```

### Create a token

To enable the action to create a pull request (`open_pr: true`) , you will need to create a new token with the correct scope. This is needed because the default `GITHUB_TOKEN` doesn't have the necessary permissions (`workflows`).

To do so, go to your GitHub account, then `Settings` -> `Developer settings` -> `Personal access tokens` -> `Fine-grained tokens` -> `Generate new token`.

Name the Token as `FRIZBEE_TOKEN` and give it a description and an expiration date.

You can then assign access to All repositories or only to specific repositories.

Should you select specific repositories, you will need to add the repository
where you are using the action.

Make sure the following scopes are assigned:

* The `workflows` scope and provide read and write access.
* The `contents` scope and provide read and write access.
* The `pull_requests` scope and provide read and write access.

### Set up the Secret

Head to the repository where you are using the action, then `Settings` -> `Secrets and variables` -> `Actions` -> `New repository secret`.

Name the secret `FRIZBEE_TOKEN` and paste the token you created in the previous
step and select `Add secret`.

#### I don't have time for that

If you prefer to automate all of this, we recommend trying out [Minder cloud](https://cloud.stacklok.com), a free for Open Source SaaS solution built on the open source project Minder. This
way you can easily automate the process of pinning your actions and container
images. You can also do a lot more than that, like monitoring your dependencies,
scanning your code for vulnerabilities, and securing configuring your
repositories and GitHub Actions.

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

## Contributing

We welcome contributions to Frizbee Action. Please see our [Contributing](./CONTRIBUTING.md) guide for more information.

## Minder

Frizbee is also a feature of the [Minder](https://github.com/stacklok/mimder) Open Source project. 
With Minder, you can also easily enable the automation of pinning your actions
and container images. You can also do a lot more, such as monitoring your
dependencies, scanning your code for vulnerabilities, and securing configuring
your repositories and GitHub Actions.

Stacklok hosts a free (forever to open source projects) SaaS service available at [Minder Cloud](https://cloud.stacklok.com). 

## License

Frizbee is licensed under the [Apache 2.0 License](./LICENSE).
