package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"
)


type Node struct {
	id int
	edge [2]*Node
}

func createNode(id int, edge0 *Node, edge1 *Node) *Node  {
	n := new(Node)
	n.id = id
	n.edge = [2]*Node{edge0, edge1}
	return n
}

const NODEHASH_SIZE = 10

func hashIndex(a *Node, b *Node) int {
	index := 0
	if a != nil {
		index += a.id * NODEHASH_SIZE
	}
	if b != nil {
		index += b.id
	}
	return index % NODEHASH_SIZE
}

type NodeHash struct {
	el *Node
	next *NodeHash
}

type NodesHash struct {
	elements [NODEHASH_SIZE]*NodeHash
}

func (h *NodesHash) getSameKey(a *Node, b *Node) *NodeHash {
	if a == nil && b == nil{
		return nil
	}
	return h.elements[hashIndex(a, b)]
}


func (h *NodesHash) get(a *Node, b *Node) *Node {
	n := h.elements[hashIndex(a, b)]
	if n == nil {
		return nil
	}
	for n != nil && (n.el.edge[0] != a || n.el.edge[1] != b) {
		n = n.next
	}
	if n == nil {
		return nil
	}
	return n.el
}

func (h *NodesHash) add(t *Node) {
	elNodeHashEntry := &NodeHash{t, nil}
	index := hashIndex(t.edge[0], t.edge[1])
	if h.elements[index] == nil {
		h.elements[index] = elNodeHashEntry
	} else {
		hashEL := h.elements[index]
		for hashEL.next != nil {
			if hashEL.el.edge == t.edge {
				return
			}
			hashEL = hashEL.next
		}
		hashEL.next = elNodeHashEntry
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
	edgeNames := [2]int{-1, -1}
	for index, edge := range n.edge {
		if edge != nil {
			edgeNames[index] = edge.id
		}
	}
	return fmt.Sprint(n.id, "(", edgeNames[0], ",", edgeNames[1], ")")
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

func (first *Node) equals(second *Node) bool {
	if first == nil && second == nil {
		return true
	} else if first != nil && second != nil {
		return first.edge[0].equals(second.edge[0]) && first.edge[1].equals(second.edge[1]) && first.id == second.id
	}
	return false
}

type RobddBuilder struct {
	creationHash NodesHash
	output *Element
	inputs []*Element
	inputsSize int
	bddTrue Node
	bddFalse Node
}

func (self *RobddBuilder) build(node *Element) *Node {
	self.output = node
	self.inputs = self.output.getAllInputs()
	self.inputsSize = len(self.inputs)
	self.bddTrue.id = 1
	self.bddFalse.id = 0

	return self.buildRecursive(2)
}

func (self *RobddBuilder) mk(id int, low *Node, high *Node) *Node {
	if low == high {
		return low
	}
	n := self.creationHash.get(low, high)
	if n == nil {
		n = createNode(id, low, high)
		self.creationHash.add(n)
	}
	return n
}

func (self *RobddBuilder) buildRecursive(i int) *Node {
	if i - 2 >= self.inputsSize {
		if self.output.eval() {
			return &self.bddTrue
		} else {
			return &self.bddFalse
		}
	} else {
		self.inputs[i - 2].val = false
		low := self.buildRecursive(i + 1)
		self.inputs[i - 2].val = true
		high := self.buildRecursive(i + 1)
		return self.mk(i + 2, low, high)
	}
}

func main() {
	b, err := ioutil.ReadFile(os.Args[1]) // just pass the file name
	if err != nil {
		fmt.Print(err)
	}

	model := new(Model)

	start := time.Now()
	model = pars(string(b))
	fmt.Println(time.Since(start))
	model.outputs[0].print(0)

	fmt.Println()
	fmt.Println()
	bdd := new(RobddBuilder)
	n := bdd.build(model.outputs[1])
	n.PrintTree()
}