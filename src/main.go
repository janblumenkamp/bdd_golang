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

const NODEHASH_SIZE = 8

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
	n := h.elements[h.hashIndex(t)]
	for n.a != t.a && n.b != t.b && n.next != nil {
		n = n.next
	}
	return n
}

func (h *NodeTupleHash) add(t *NodeTuple) {
	var index = h.hashIndex(t)
	if h.elements[index] == nil {
		h.elements[index] = t
	} else {
		var el = h.elements[index]
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

func (h *NodeHash) getSameKey(t *Node) *Node {
	if t == nil {
		return nil
	}
	return h.elements[h.hashIndex(t)]
}

func (h *NodeHash) get(t *Node) *Node {
	if t == nil {
		return nil
	}
	n := h.elements[h.hashIndex(t)]
	for n.edge == t.edge && n.next != nil {
		n = n.next
	}
	return n
}

func (h *NodeHash) hashIndex(t *Node) int {
	var index_a = int(uintptr(unsafe.Pointer(t.edge[0])))
	var index_b = int(uintptr(unsafe.Pointer(t.edge[1])))
	return (index_a * (NODEHASH_SIZE / 2) + index_b) % NODEHASH_SIZE
}

func (h *NodeHash) contains(t *Node) bool {
	var el = h.elements[h.hashIndex(t)]
	for el != nil {
		if el.edge == t.edge {
			return true
		}
		el = el.next
	}
	return false
}

func (h *NodeHash) add(t *Node) {
	var index = h.hashIndex(t)
	if h.elements[index] == nil {
		h.elements[index] = t
	} else {
		var el = h.elements[index]
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

func (n1 *Node) product(n2 *Node) []*Node {
	start := NodeTuple{createNode(n1.name + n2.name, nil, nil),n1, n2, nil}
	queue := append(make([]*NodeTuple, 0), &start)
	queueProduct := append(make([]*Node, 0), start.node) // Needed for the minimization
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
					queueProduct = append(queueProduct, succ.node)
					x.node.edge[i] = succ.node
				} else {
					succ = *hash.get(&succ)
					x.node.edge[i] = succ.node
				}
			}
		}
	}
	return queueProduct
}

func minimize(generizationQueue []*Node) *Node {
	hashMin := NodeHash{}
	for i := len(generizationQueue) - 1; i >= 0; i-- {
		flag := false
		q := generizationQueue[i]

		// Ignore child nodes not leading to end (will be removed when parent node is processed)
		if q.edge[0] == nil && q.edge[1] == nil && !q.final {
			continue
		}

		// Remove child nodes not leading to final state
		for j := 0; j < 2; j++ {
			if q.edge[j] != nil && !q.edge[j].final {
				if q.edge[j].edge[0] == nil && q.edge[j].edge[1] == nil {
					q.edge[j] = nil
				}
			}
		}

		// If the current node is in an equivalency class with any other then don't insert into global state
		var n *Node
		for n = hashMin.getSameKey(q); n != nil; n = n.next {
			edge0equiv := q.edge[0]
			if edge0equiv != nil {
				edge0equiv = edge0equiv.min_equiv
			}
			edge1equiv := q.edge[1]
			if edge1equiv != nil {
				edge1equiv = edge1equiv.min_equiv
			}
			if n.edge[0] == edge0equiv && n.edge[1] == edge1equiv && !n.final {
				flag = true
				break
			}
		}

		if !flag {
			edge0Next := q.edge[0]
			if edge0Next != nil {
				edge0Next = edge0Next.min_equiv
			}
			edge1Next := q.edge[1]
			if edge1Next != nil {
				edge1Next = edge1Next.min_equiv
			}
			n = createNode(q.name + "_c", edge0Next, edge1Next)
			if i == 0 {
				return n
			}
			n.final = q.final
			n.min_equiv = q

			hashMin.add(n)
		}
		q.min_equiv = n
	}
	return nil
}

func (n1 *Node) unify(n2 *Node) *Node {
	return minimize(n1.product(n2));
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

	q21 := createNode("q6q13_c", nil, nil)
	q21.final = true
	q20 := createNode("q5q11_c", q21, q21)
	q19 := createNode("q5q12_c", q21, nil)
	q16 := createNode("q3q9_c", q19, q20)
	q15 := createNode("q2q8_c", q19, nil)
	q14 := createNode("q1q7_c", q15, q16)

	unified := q1.unify(q7)
	unified.PrintTree()

	if !q14.equals(unified) {
		fmt.Println("the generated tree does not equal the minimized tree")
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

	q12 := createNode("q8q8_c", nil, nil)
	q12.final = true
	q11 := createNode("q7q7_c", q12, nil)
	q10 := createNode("q3q3_c", q11, q11)
	q9 := createNode("q1q1_c", q10, q10)

	if !q1.equals(q1) {
		fmt.Println("The tree is not equal to itself")
	}

	unified := q1.unify(q1)
	unified.PrintTree()

	if !q9.equals(unified) {
		fmt.Println("the generated tree does not equal the minimized tree")
	}
}


func main() {
	TestTreeFromPaper()
	TestTreeWithFourIsomorph()
}