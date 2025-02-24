/*
Copyright (c) 2012-2013 Fredrik Ehnbom
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this
   list of conditions and the following disclaimer.
2. Redistributions in binary form must reproduce the above copyright notice,
   this list of conditions and the following disclaimer in the documentation
   and/or other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
(INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

package parser

import (
	"bytes"
	"fmt"
	"github.com/jxo/lime/text"
)

type (
	// Generic DataSource interface
	DataSource interface {
		// Returns the data between the start and end indices
		Data(start, end int) string
	}

	// Node is the base structure used to represent a directed acyclic graph
	Node struct {
		// The Range this node occupies, as referenced to the DataSource.
		Range text.Region
		// The Name of this node.
		Name string
		// The Children of this Node.
		Children []*Node
		// The DataSource to query when Node.Data is called.
		P DataSource
	}
)

// format is a helper function used by String for recursively
// indenting and adding string data to the provided "buf".
func (n *Node) format(buf *bytes.Buffer, indent string) {
	buf.WriteString(indent)
	buf.WriteString(fmt.Sprintf("%d-%d", n.Range.Begin(), n.Range.End()))
	buf.WriteString(": \"")
	buf.WriteString(n.Name)
	buf.WriteString("\"")
	if len(n.Children) == 0 {
		buf.WriteString(" - Data: \"")
		buf.WriteString(n.Data())
		buf.WriteString("\"\n")
		return
	}
	buf.WriteRune('\n')
	indent += "\t"
	for _, child := range n.Children {
		child.format(buf, indent)
	}
}

// Queries the DataSource and returns the data for this Node's Range.
func (n *Node) Data() string {
	return n.P.Data(n.Range.Begin(), n.Range.End())
}

// String-function to satisfy the fmt.Stringer interface.
// Returns an indented string representation of this node
// and its sub-tree.
//
// To get the Data contained within this node, use #Data instead.
func (n *Node) String() string {
	buf := bytes.NewBuffer(nil)
	n.format(buf, "")
	return buf.String()
}

// Discards child nodes whose Range starts after "pos"
func (n *Node) Discard(pos int) {
	back := len(n.Children)
	popIdx := 0
	for i := back - 1; i >= 0; i-- {
		node := n.Children[i]
		if node.Range.End() <= pos {
			popIdx = i + 1
			break
		}
	}
	if popIdx != back {
		n.Children = n.Children[:popIdx]
	}
}

// Cleanup is different from Discard in that
// it returns a new Node containing the Children whose
// Range are within the region of "pos" and "end".
//
// The receiver Node "n" will be the same as if
// we had called n.Discard(pos).
//
// Child-nodes starting after "end" will be in
// neither "n" nor the new returned Node.
func (n *Node) Cleanup(pos, end int) *Node {
	var popped Node
	popped.Range = text.Region{pos, end}
	back := len(n.Children)
	popIdx := 0
	popEnd := back
	if end == 0 {
		end = -1
	}
	if pos == 0 {
		pos = -1
	}

	for i := back - 1; i >= 0; i-- {
		node := n.Children[i]
		if node.Range.End() <= pos {
			popIdx = i + 1
			break
		}
		if node.Range.Begin() > end {
			popEnd = i + 1
		}
	}

	if popEnd != 0 {
		popped.Children = n.Children[popIdx:popEnd]
		c := make([]*Node, len(popped.Children))
		copy(c, popped.Children)
		popped.Children = c
	}
	if popIdx != back {
		n.Children = n.Children[:popIdx]
	}
	return &popped
}

// Clones this node-sub tree
func (n *Node) Clone() *Node {
	ret := *n
	ret.Children = make([]*Node, len(n.Children))
	for i := range n.Children {
		ret.Children[i] = n.Children[i].Clone()
	}
	return &ret
}

// Adjusts this node's range and that of its children at "position" by "delta".
func (n *Node) Adjust(position, delta int) {
	n.Range.Adjust(position, delta)
	for _, child := range n.Children {
		child.Adjust(position, delta)
	}
}

// UpdateRange makes sure that all parent nodes ranges
// contain their children.
func (n *Node) UpdateRange() text.Region {
	for _, child := range n.Children {
		curr := child.UpdateRange()
		if curr.Begin() < n.Range.A {
			n.Range.A = curr.Begin()
		}
		if curr.End() > n.Range.B {
			n.Range.B = curr.End()
		}
	}
	return n.Range
}

// Append node "child" at the end of this node's Children slice.
func (n *Node) Append(child *Node) {
	n.Children = append(n.Children, child)
}

// Simplify this sub-tree by merging children into the parent where
// the parent only has a single child and the parent and child occupy
// the exact same Range.
func (n *Node) Simplify() {
	for _, child := range n.Children {
		child.Simplify()
	}
	if len(n.Children) == 1 && n.Children[0].Range == n.Range {
		*n = *n.Children[0]
	}
}
