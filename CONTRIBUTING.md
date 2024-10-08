**Table of Contents**

- [Contributing guidelines](#contributing-guidelines)
  - [Terms](#terms)
  - [Certificate of Origin](#certificate-of-origin)
  - [DCO Sign Off](#dco-sign-off)
  - [Code of Conduct](#code-of-conduct)
  - [Contributing a patch](#contributing-a-patch)
  - [Issue and pull request management](#issue-and-pull-request-management)
  - [Pre-check before submitting a PR](#pre-check-before-submitting-a-pr)

# Contributing guidelines

## Terms

All contributions to the repository must be submitted under the terms of the
[Apache Public License 2.0](https://www.apache.org/licenses/LICENSE-2.0).

## Certificate of Origin

By contributing to this project, you agree to the Developer Certificate of Origin (DCO). This
document was created by the Linux Kernel community and is a simple statement that you, as a
contributor, have the legal right to make the contribution. See the
[DCO](https://github.com/open-cluster-management-io/community/blob/main/DCO) file for details.

## DCO Sign Off

You must sign off your commit to state that you certify the
[DCO](https://github.com/open-cluster-management-io/community/blob/main/DCO). To certify your commit
for DCO, add a line like the following at the end of your commit message:

```
Signed-off-by: John Smith <john@example.com>
```

This can be done with the `--signoff` option to `git commit`. See the
[Git documentation](https://git-scm.com/docs/git-commit#Documentation/git-commit.txt--s) for
details.

## Code of Conduct

The Open Cluster Management project has adopted the CNCF Code of Conduct. Refer to our
[Community Code of Conduct](https://github.com/open-cluster-management-io/community/blob/main/CODE_OF_CONDUCT.md)
for details.

## Contributing a patch

1. Submit an issue describing your proposed change to the repository in question. The repository
   owners will respond to your issue promptly.
2. Fork the desired repository, then develop and test your code changes.
3. Submit a pull request.

In order to contribute a binary to `acm-cli`, add a new line to
[`build/cli_map.csv`](build/cli_map.csv). The file is in CSV (Comma Separated Values) format with
the following columns:

- Git URL
- Build command
- Build output directory

So the new line would be formatted as:

```
<git-url>,<build-command>,<build-output-directory>
```

The expectation is that the output binary filenames in the build output directory are formatted as
`<os>-<arch>-<binary_name>`. The CLI packaging script greedily strips all characters before the
final dash ('-') to determine the filename of the binary inside the packaged archive and packages it
with the Apache license from this repository.

## Issue and pull request management

Anyone can comment on issues and submit reviews for pull requests. In order to be assigned an issue
or pull request, you can leave a `/assign <your Github ID>` comment on the issue or pull request
(PR).

## Pre-check before submitting a PR

Before submitting a PR, please perform the following steps:

```shell
make fmt
make lint
make build-image
make e2e-test
```
