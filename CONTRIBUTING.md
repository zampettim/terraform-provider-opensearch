# Contributing Guidelines

Thank you for your interest in contributing to our project. Whether it's a bug report, new feature, correction, or additional
documentation, we greatly value feedback and contributions from our community.

Please read through this document before submitting any issues or pull requests to ensure we have all the necessary
information to effectively respond to your bug report or contribution.


## Reporting Bugs/Feature Requests

We welcome you to use the GitHub issue tracker to report bugs or suggest features.

When filing an issue, please check existing open, or recently closed, issues to make sure somebody else hasn't already
reported the issue. Please try to include as much information as you can. Details like these are incredibly useful:

* A reproducible test case or series of steps
* The version of our code being used
* Any modifications you've made relevant to the bug
* Anything unusual about your environment or deployment


## Contributing via Pull Requests
Contributions via pull requests are much appreciated. Before sending us a pull request, please ensure that:

1. You are working against the latest source on the *main* branch.
2. You check existing open, and recently merged, pull requests to make sure someone else hasn't addressed the problem already.
3. You open an issue to discuss any significant work - we would hate for your time to be wasted.

To send us a pull request, please:

1. Fork the repository.
2. Modify the source; please focus on the specific change you are contributing. If you also reformat all the code, it will be hard for us to focus on your change.
3. Ensure local tests pass. See [Local Development Setup](#local-development-setup) below for prerequisites.
4. Commit to your fork using clear commit messages.
5. Send us a pull request, answering any default questions in the pull request interface.
6. Pay attention to any automated CI failures reported in the pull request, and stay involved in the conversation.

GitHub provides additional document on [forking a repository](https://help.github.com/articles/fork-a-repo/) and
[creating a pull request](https://help.github.com/articles/creating-a-pull-request/).


## Generate documentation using Tfplugindocs
1. Install [Tfplugindocs](https://github.com/hashicorp/terraform-plugin-docs)
2. Run tfplugindocs generate


## Finding contributions to work on
Looking at the existing issues is a great way to find something to contribute on. As our projects, by default, use the default GitHub issue labels (enhancement/bug/duplicate/help wanted/invalid/question/wontfix), looking at any 'help wanted' issues is a great place to start.


## Code of Conduct
This project has adopted the [Amazon Open Source Code of Conduct](https://aws.github.io/code-of-conduct).
For more information see the [Code of Conduct FAQ](https://aws.github.io/code-of-conduct-faq) or contact
opensource-codeofconduct@amazon.com with any additional questions or comments.


## Security issue notifications
If you discover a potential security issue in this project we ask that you notify AWS/Amazon Security via our [vulnerability reporting page](http://aws.amazon.com/security/vulnerability-reporting/). Please do **not** create a public github issue.


## Local Development Setup

To run the full test suite locally before submitting a pull request, the following tools are required:

### Common Prerequisites (All Platforms)

- [Go](https://golang.org/doc/install) (see `go.mod` for required version)
- [Terraform](https://developer.hashicorp.com/terraform/install)
- [Docker](https://docs.docker.com/get-docker/) with Docker Compose support
- `make` (usually pre-installed on macOS/Linux)

### macOS-Specific Prerequisites

The CI workflow uses Linux-specific utilities that are **not installed by default on macOS**. You must install the following via [Homebrew](https://brew.sh/):

```bash
# Required: GNU coreutils provides 'timeout', used by the wait-for-endpoint script
brew install coreutils

# Recommended for full CI parity
brew install golangci-lint    # v2.x (matches CI)
brew install goreleaser       # For release config validation
```

After installing `coreutils`, create a symlink so the `wait-for-endpoint` script can find `timeout`:

```bash
sudo ln -s $(which gtimeout) /usr/local/bin/timeout
```

> **Note:** Without `timeout`, the `make wait` and `make ci-test*` targets will fail with `timeout: command not found` even if OpenSearch is healthy.

> **Note on golangci-lint:** The project uses `golangci-lint` **v2.x**. If you have a legacy v1 configuration file (`.golangci.yml` without `version: "2"`), run `golangci-lint migrate` to auto-convert it to the v2 format.

### Linux-Specific Notes

On Linux, you may need to increase the kernel parameter `vm.max_map_count` before starting OpenSearch:

```bash
sudo sysctl -w vm.max_map_count=262144
```

macOS Docker Desktop handles this automatically, so no extra step is needed on Mac.

### Running Tests Locally

A `Makefile` is provided at the repository root to replicate CI logic. Common commands:

| Command | Purpose |
|---------|---------|
| `make check` | Fast pre-commit checks (lint + `go mod tidy` + `terraform fmt`) |
| `make ci-test-os2` | Full CI simulation against OpenSearch 2.x |
| `make ci-test-os3` | Full CI simulation against OpenSearch 3.x |
| `make help` | Show all available targets and descriptions |

Run `make help` for the full list of available targets.

## Licensing

See the [LICENSE](LICENSE) file for our project's licensing. We will ask you to confirm the licensing of your contribution.
