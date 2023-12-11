# `github.com/DataDog/appsec-internal-go`

This repository hosts a go module that provide shared implementations for
internal details of various DataDog libraries and agents. This module is not
intended to be used directly by end-users.

## Updating embedded Rules

Embedded rules (at `appsec/rules.json`) are updated by the
`_tools/rules-updater/update.sh` script. A GitHub Workflow named
`Update AppSec Rules` can be used to perform this task entirely from the GitHub
Web UI.
