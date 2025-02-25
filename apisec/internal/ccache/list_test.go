// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023-present Datadog, Inc.

package ccache

import (
	"math/rand"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestList(t *testing.T) {
	t.Run("Insert", func(t *testing.T) {
		list := newList[int]()
		assertList(t, list)

		list.Insert(1)
		assertList(t, list, 1)

		list.Insert(2)
		assertList(t, list, 1, 2)

		list.Insert(3)
		assertList(t, list, 1, 2, 3)
	})

	t.Run("Remove", func(t *testing.T) {
		list := newList[int]()
		assertList(t, list)

		node := list.Insert(1337)
		list.Remove(node)
		assertList(t, list)

		nodes := make([]*listNode[int], 5)
		values := make([]int, 5)
		for i := range len(nodes) {
			nodes[i] = list.Insert(i)
			values[i] = i
		}

		for _, i := range rand.Perm(len(nodes)) {
			list.Remove(nodes[i])
			values = slices.DeleteFunc(values, func(v int) bool { return v == i })
			assertList(t, list, values...)
		}
	})

	t.Run("MoveToFront", func(t *testing.T) {
		list := newList[int]()

		items := make([]*listNode[int], 5)
		for i := range 5 {
			items[i] = list.Insert(i)
		}
		assertList(t, list, 0, 1, 2, 3, 4)

		list.MoveToFront(items[0])
		assertList(t, list, 1, 2, 3, 4, 0)

		list.MoveToFront(items[1])
		assertList(t, list, 2, 3, 4, 0, 1)
	})
}

func assertList[T any](t *testing.T, list *linkedList[T], values ...T) {
	t.Helper()

	if len(values) == 0 {
		assert.Nil(t, list.Head)
		assert.Nil(t, list.Tail)
		return
	}

	// Insertions are done at the Head, so we list from the tail here...
	node := list.Tail
	var lastNode *listNode[T]
	for _, expected := range values {
		lastNode = node
		require.NotNil(t, node)
		assert.Equal(t, node.Value, expected)
		node = node.Prev
	}

	assert.Equal(t, lastNode, list.Head)
}
