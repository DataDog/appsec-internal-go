// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023-present Datadog, Inc.

package ccache

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBucket(t *testing.T) {
	t.Run("Get", func(t *testing.T) {
		t.Run("Hit", func(t *testing.T) {
			subject := testBucket()
			item := subject.get(1337)
			require.NotNil(t, item)
			assert.Equal(t, "elite", item.Value())
		})

		t.Run("Miss", func(t *testing.T) {
			subject := testBucket()
			assert.Nil(t, subject.get(42))
		})
	})

	t.Run("Delete", func(t *testing.T) {
		subject := testBucket()
		assert.NotNil(t, subject.delete(1337))
		assert.Nil(t, subject.get(1337))
	})

	t.Run("Set", func(t *testing.T) {
		t.Run("New", func(t *testing.T) {
			subject := testBucket()
			item, oldItem := subject.set(42, "purpose of life")
			require.NotNil(t, item)
			assert.Equal(t, "purpose of life", item.Value())
			assert.Nil(t, oldItem)

			got := subject.get(42)
			assert.Equal(t, item, got)
		})

		t.Run("Existing", func(t *testing.T) {
			subject := testBucket()
			item, oldItem := subject.set(1337, "goat")
			require.NotNil(t, item)
			assert.Equal(t, "goat", item.Value())
			require.NotNil(t, oldItem)
			assert.Equal(t, "elite", oldItem.Value())

			got := subject.get(1337)
			assert.Equal(t, item, got)
		})
	})
}

func testBucket() *bucket[int, string] {
	return &bucket[int, string]{
		lookup: map[int]*Item[int, string]{
			1337: {key: 1337, value: "elite"},
		},
	}
}
