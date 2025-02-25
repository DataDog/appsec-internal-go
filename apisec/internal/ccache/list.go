// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023-present Datadog, Inc.

package ccache

type (
	// linkedList is a simple doubly linked list implementation.
	linkedList[T any] struct {
		Head *listNode[T]
		Tail *listNode[T]
	}

	// listNode is a node in a [linkedList].
	listNode[T any] struct {
		Next  *listNode[T]
		Prev  *listNode[T]
		Value T
	}
)

// newList creates a new, empty [linkedList].
func newList[T any]() *linkedList[T] {
	return &linkedList[T]{}
}

// Insert adds a new node to the front of the list, and returns it.
func (l *linkedList[T]) Insert(value T) *listNode[T] {
	node := &listNode[T]{Value: value}
	l.nodeToFront(node)
	return node
}

// MoveToFront moves the specified node to the front of the list.
func (l *linkedList[T]) MoveToFront(node *listNode[T]) {
	l.Remove(node)
	l.nodeToFront(node)
}

// Remove removes the specified node from the list.
func (l *linkedList[T]) Remove(node *listNode[T]) {
	next := node.Next
	prev := node.Prev

	if next == nil {
		l.Tail = node.Prev
	} else {
		next.Prev = prev
	}

	if prev == nil {
		l.Head = node.Next
	} else {
		prev.Next = next
	}

	node.Next = nil
	node.Prev = nil
}

// nodeToFront places the specified node at the front of the list.
func (l *linkedList[T]) nodeToFront(node *listNode[T]) {
	head := l.Head
	l.Head = node
	if head == nil {
		l.Tail = node
		return
	}
	node.Next = head
	head.Prev = node
}
