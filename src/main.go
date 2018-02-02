package main

import (
	"fmt"
	"unsafe"
)

type Node struct {
	name string
	edge [2]*Node
	final bool
	next *Node
	min_equiv *Node // Corresponding node in minimization
}

func createEmptyNode(name string) *Node {
	n := new(Node)
	n.name = name
	n.edge = [2]*Node{nil, nil}
	n.final = false
	n.next = nil
	return n
}

func createNode(name string, edge0 *Node, edge1 *Node) *Node  {
	n := new(Node)
	n.name = name
	n.edge = [2]*Node{edge0, edge1}
	n.final = false
	n.next = nil
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

const NODEHASH_SIZE = 10

type NodeTupleHash struct {
	elements [NODEHASH_SIZE]*NodeTuple
}

func hashIndex(a *Node, b *Node) int {
	index_a := int(uintptr(unsafe.Pointer(a))) >> 6
	index_b := int(uintptr(unsafe.Pointer(b))) >> 6
	result := (index_a * (NODEHASH_SIZE / 2) + index_b) % NODEHASH_SIZE
	return result
}

func (h *NodeTupleHash) get(a *Node, b *Node) *NodeTuple {
	n := h.elements[hashIndex(a, b)]
	if n == nil {
		return nil
	}
	for n != nil && n.a != a && n.b != b {
		n = n.next
	}
	return n
}

func (h *NodeTupleHash) add(t *NodeTuple) {
	index := hashIndex(t.a, t.b)
	if h.elements[index] == nil {
		h.elements[index] = t
	} else {
		el := h.elements[index]
		for el.next != nil {
			if el.a == t.a && el.b == t.b {
				return
			}
			el = el.next
		}
		el.next = t
	}
}

type NodeHash struct {
	elements [NODEHASH_SIZE]*Node
}

func (h *NodeHash) getSameKey(a *Node, b *Node) *Node {
	if a == nil && b == nil{
		return nil
	}
	return h.elements[hashIndex(a, b)]
}

func (h *NodeHash) add(t *Node) {
	index := hashIndex(t.edge[0], t.edge[1])
	if h.elements[index] == nil {
		h.elements[index] = t
	} else {
		el := h.elements[index]
		for el.next != nil {
			if el.edge == t.edge {
				return
			}
			el = el.next
		}
		el.next = t
	}
}

type NodeLabeled struct {
	*Node
	level int
}

type NodeLabeledQueue []*NodeLabeled

func (labeledNodes NodeLabeledQueue) contains(a *Node) bool {
	for _, b := range labeledNodes {
		if b.Node == a {
			return true
		}
	}
	return false
}

func (n *Node) String() string {
	if n == nil {
		return ""
	}
	edgeNames := [2]string{"", ""}
	for index, edge := range n.edge {
		if edge != nil {
			edgeNames[index] = edge.name
		}
	}
	return fmt.Sprint(n.name, "(", edgeNames[0], ",", edgeNames[1], ")")
}

func (n *Node) PrintTree() {
	processQueue := append(make(NodeLabeledQueue, 0), &NodeLabeled{n, 0})

	previousLevel := 0

	for len(processQueue) > 0 {
		node := processQueue[0] // Pop
		processQueue = processQueue[1:]

		if previousLevel < node.level {
			previousLevel = node.level
			fmt.Println()
		}

		fmt.Print(node)

		for _, edge := range node.edge {
			if edge != nil && !processQueue.contains(edge) {
				new_node := &NodeLabeled{edge, node.level + 1}
				processQueue = append(processQueue, new_node)
			}
		}
	}
	fmt.Println()
	fmt.Println()
}

func (n1 *Node) operation(n2 *Node, op func(a *Node, b *Node) bool, isFinal func(a *Node, b *Node) bool) []*Node {
	start := NodeTuple{createNode(n1.name + n2.name, nil, nil),n1, n2, nil}
	queue := append(make([]*NodeTuple, 0), &start)
	queueProduct := append(make([]*Node, 0), start.node) // Needed for the minimization
	hash := NodeTupleHash{}
	for len(queue) > 0 {
		x := queue[0] // head
		queue = queue[1:] // remove
		for i := 0; i < 2; i++ {
			if op(x.a.edge[i], x.b.edge[i]) {
				succ := hash.get(x.a.edge[i], x.b.edge[i])
				if succ == nil {
					succ = &NodeTuple{createEmptyNode(x.a.edge[i].name + x.b.edge[i].name), x.a.edge[i], x.b.edge[i], nil}
					hash.add(succ)
					queue = append(queue, succ)
					succ.node.final = isFinal(succ.a, succ.b)
					queueProduct = append(queueProduct, succ.node)
				}
				x.node.edge[i] = succ.node
			}
		}
	}
	return queueProduct
}

func minimize(generizationQueue []*Node) *Node {
	hashMin := NodeHash{}
	var n *Node
	for i := len(generizationQueue) - 1; i >= 0; i-- {
		flag := false
		q := generizationQueue[i]

		// Ignore child nodes not leading to end
		if q.edge[0] == nil && q.edge[1] == nil && !q.final {
			continue
		}

		// If the current node is in an equivalency class with any other then don't insert into global state
		edgeEquiv := q.edge
		for i := 0; i < 2; i++ {
			if edgeEquiv[i] != nil {
				edgeEquiv[i] = edgeEquiv[i].min_equiv
			}
		}
		for n = hashMin.getSameKey(edgeEquiv[0], edgeEquiv[1]); n != nil; n = n.next {
			if n.edge == edgeEquiv && !n.final {
				flag = true
				break
			}
		}

		if !flag {
			n = createNode(q.name, edgeEquiv[0], edgeEquiv[1])
			n.final = q.final

			hashMin.add(n) // NEEDS to be n, otherwise already merged states would not be considered
		}

		// Merge isomorphic nodes
		if n.edge[0] != nil && n.edge[0] == n.edge[1] {
			n = n.edge[0]
		}

		q.min_equiv = n
	}

	return n
}

func andOp(a *Node, b *Node) bool {
	return a != nil && b != nil
}

func andFin(a *Node, b *Node) bool {
	return a.final && b.final
}

func (n1 *Node) and(n2 *Node) *Node {
	return minimize(n1.operation(n2, andOp, andFin))
}

func (first *Node) equals(second *Node) bool {
	if first == nil && second == nil {
		return true
	} else if first != nil && second != nil {
		return first.edge[0].equals(second.edge[0]) && first.edge[1].equals(second.edge[1]) && first.name == second.name
	}
	return false
}

func TestTreeFromPaper() {
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

	q21 := createNode("q6q13", nil, nil)
	q21.final = true
	q19 := createNode("q5q12", q21, nil)
	q16 := createNode("q3q9", q19, q21)
	q15 := createNode("q2q8", q19, nil)
	q14 := createNode("q1q7", q15, q16)

	unified := q1.and(q7)

	if !q14.equals(unified) {
		fmt.Println("the generated tree does not equal the minimized tree")
		unified.PrintTree()
	}
}

func TestTreeWithFourIsomorph() {
	q8 := createNode("q8", nil, nil)
	q8.final = true
	q7 := createNode("q7", q8, nil)
	q6 := createNode("q6", q8, nil)
	q5 := createNode("q5", q8, nil)
	q4 := createNode("q4", q8, nil)
	q3 := createNode("q3", q6, q7)
	q2 := createNode("q2", q4, q5)
	q1 := createNode("q1", q2, q3)

	q10 := createNode("q8q8", nil, nil)
	q10.final = true
	q9 := createNode("q7q7", q10, nil)

	if !q1.equals(q1) {
		fmt.Println("The tree is not equal to itself")
	}

	unified := q1.and(q1)

	if !q9.equals(unified) {
		fmt.Println("the generated tree does not equal the minimized tree")
		unified.PrintTree()
	}
}

func main() {
	TestTreeFromPaper()
	TestTreeWithFourIsomorph()
}