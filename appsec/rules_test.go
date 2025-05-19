// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023-present Datadog, Inc.

package appsec

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultRuleset(t *testing.T) {
	rules, err := DefaultRuleset()
	require.NoError(t, err)
	require.NotEmpty(t, rules)
}

func TestDefaultRulesetMap(t *testing.T) {
	rules, err := DefaultRulesetMap()
	require.NoError(t, err)
	require.NotEmpty(t, rules)
}
