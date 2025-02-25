// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023-present Datadog, Inc.

package ccache

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestItem(t *testing.T) {
	t.Run("Key", func(t *testing.T) {
		i := newItem(1337, "elite")
		require.Equal(t, 1337, i.Key())
	})

	t.Run("Value", func(t *testing.T) {
		i := newItem(1337, "elite")
		require.Equal(t, "elite", i.Value())
	})
}
