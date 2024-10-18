#!/bin/bash

# Unless explicitly stated otherwise all files in this repository are licensed
# under the Apache License Version 2.0.
# This product includes software developed at Datadog (https://www.datadoghq.com/).
# Copyright 2022 Datadog, Inc.
#

# Generates the rule.go file using the recommended rules for the specified tag version
# Usage: ./update.sh <tag>
# Example: ./update.sh 1.2.5
#
# When the tag specified is "latest", the latest version of the rules file is
# determined using the GitHub API.

set -eu

[ $# -ne 1 ] && echo "Usage: $0 \"version\"" >&2 && exit 1

if [ -z ${GITHUB_TOKEN:-} ]; then
  echo "The GITHUB_TOKEN environmnent variable must be set to a valid GitHub"
  echo "token with read access to the DataDog/appsec-event-rules repository."
  exit 1
fi

echo "================ Minifying ================"

tmpDir="$(mktemp -d /tmp/rule-update-XXXXXXXXX)"
scriptPath="$(readlink -f $0)"
scriptDir="$(dirname $scriptPath)"
destDir="$(readlink -f "$scriptDir/../../appsec/")"

trap "rm -r $tmpDir" EXIT

DOCKER_BUILDKIT=1 docker build -o type=local,dest="$tmpDir" --build-arg version="$1" --secret "id=GITHUB_TOKEN,env=GITHUB_TOKEN" --no-cache "$scriptDir"
echo "================   Done    ================"
cp -v $tmpDir/embed.go $tmpDir/rules.json "$destDir"
echo "Output written to $destDir"
