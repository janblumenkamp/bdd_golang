package main

import (
	"fmt"
	"unsafe"
)

type Node struct {
	name string
	edge [2]*Node
	final bool
}

func createEmptyNode(name string) *Node {
	n := new(Node)
	n.name = name
	n.edge = [2]*Node{nil, nil}
	n.final = false
	return n
}

func createNode(name string, edge0 *Node, edge1 *Node) *Node  {
	n := new(Node)
	n.name = name
	n.edge = [2]*Node{edge0, edge1}
	n.final = false
	return n
}

func (n *Node) isFinal() bool  {
	if n == nil {
		return false
	}
	return n.final
}

type NodeTuple struct {
	node *Node
	a *Node
	b *Node
	next *NodeTuple // relevant for hashing
}

const NODEHASH_SIZE = 1024

type NodeTupleHash struct {
	elements [NODEHASH_SIZE]*NodeTuple
}

func (h *NodeTupleHash) hashIndex(t *NodeTuple) int {
	var index_a = int(uintptr(unsafe.Pointer(t.a)))
	var index_b = int(uintptr(unsafe.Pointer(t.b)))
	return (index_a * (NODEHASH_SIZE / 2) + index_b) % NODEHASH_SIZE
}

func (h *NodeTupleHash) contains(t *NodeTuple) bool {
	var el = h.elements[h.hashIndex(t)]
	for el != nil {
		if el.a == t.a && el.b == t.b {
			return true
		}
		el = el.next
	}
	return false
}

func (h *NodeTupleHash) get(t *NodeTuple) *NodeTuple {
	return h.elements[h.hashIndex(t)]
}

func (h *NodeTupleHash) add(t *NodeTuple) {
	var index = h.hashIndex(t)
	if h.elements[index] == nil {
		h.elements[index] = t
	} else {
		var el = h.elements[index]
		for el.next != nil {
			el = el.next
		}
		el.next = t
	}
}

type NodeBFS struct {
	node *Node
	level int
}

func contains(a *Node, list []*NodeBFS) bool {
	for _, b := range list {
		if b.node == a {
			return true
		}
	}
	return false
}

func PrintTree(n *Node) {
	queue := make([]*NodeBFS, 0)
	queue = append(queue, &NodeBFS{n, 0})
	previouslevel := 0
	for len(queue) > 0 {
		x := queue[0]
		if previouslevel != x.level {
			previouslevel = x.level
			fmt.Println()
		}
		if x.node.final {
			fmt.Print("final ")
		}
		fmt.Print("(", x.node.name, "->")
		if x.node.edge[0] != nil && x.node.edge[0] == x.node.edge[1] {
			fmt.Print("10: ", x.node.edge[0].name)
		} else {
			if x.node.edge[0] != nil {
				fmt.Print("0: ", x.node.edge[0].name, " ")
			}
			if x.node.edge[1] != nil {
				fmt.Print("1: ", x.node.edge[1].name)
			}
		}
		fmt.Print(")")
		queue = queue[1:]
		if x.node.edge[0] != nil && !contains(x.node.edge[0], queue) {
			queue = append(queue, &NodeBFS{x.node.edge[0], x.level + 1})
		}
		if x.node.edge[1] != nil && !contains(x.node.edge[1], queue) {
			queue = append(queue, &NodeBFS{x.node.edge[1], x.level + 1})
		}
	}
	fmt.Println()
	fmt.Println()
}

func (n1 *Node) product(n2 *Node) *Node {
	start := NodeTuple{createNode(n1.name + n2.name, nil, nil),n1, n2, nil}
	queue := append(make([]*NodeTuple, 0), &start)
	hash := NodeTupleHash{}
	for len(queue) > 0 {
		x := queue[0] // head
		queue = queue[1:] // remove
		for i := 0; i < 2; i++ {
			succ := NodeTuple{nil,x.a.edge[i], x.b.edge[i], nil}
			if succ.a != nil && succ.b != nil {
				if !hash.contains(&succ) {
					hash.add(&succ)
					queue = append(queue, &succ)
					succ.node = createEmptyNode(x.a.edge[i].name + x.b.edge[i].name)
					succ.node.final = succ.a.final && succ.b.final
					x.node.edge[i] = succ.node
				} else {
					succ = *hash.get(&succ)
					x.node.edge[i] = succ.node
				}
			}
		}
	}
	return start.node
}

func main() {
	q6 := createNode("q6", nil, nil)
	q6.final = true
	q5 := createNode("q5", q6, q6)
	q4 := createNode("q4", q6, nil)
	q3 := createNode("q3", q5, q5)
	q2 := createNode("q2", q4, q4)
	q1 := createNode("q1", q2, q3) // start

	q13 := createNode("q13", nil, nil)
	q13.final = true
	q12 := createNode("q12", q13, nil)
	q11 := createNode("q11", q13, q13)
	q10 := createNode("q10", nil, q13)
	q9 := createNode("q9", q12, q11)
	q8 := createNode("q8", q11, q10)
	q7 := createNode("q7", q8, q9) // start

	PrintTree(q1)
	PrintTree(q7)
	PrintTree(q1.product(q7))
}