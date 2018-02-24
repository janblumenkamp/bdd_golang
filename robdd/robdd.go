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
		index += a.id * (NODEHASH_SIZE / 2)
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
	edgeNames := [2]string{"", ""}
	for index, edge := range n.edge {
		if edge != nil {
			edgeNames[index] = fmt.Sprintf("%p", edge)
		}
	}
	return fmt.Sprint(fmt.Sprintf("%p", n), "(", edgeNames[0], ",", edgeNames[1], ")")
}

func (self *RobddBuilder) getIdentifier(node *Node) string {
	if node == nil {
		return ""
	}

	if node.id == 0 {
		return "false"
	} else if node.id == 1 {
		return "true"
	} else {
		return fmt.Sprint(self.inputs[node.id - 2].name, fmt.Sprintf("_%p", node))
	}
}

func (self *RobddBuilder) StringRecursive(n *Node) string {
	if n == nil || (n.edge[0] == nil && n.edge[1] == nil) {
		return ""
	}

	//s := fmt.Sprint(self.getIdentifier(n), ",", self.getIdentifier(n.edge[0]), ",", self.getIdentifier(n.edge[1]), ";")
	ownId := self.getIdentifier(n)
	s := ""
	s += fmt.Sprint("g.addEdge(\"", ownId, "\", \"", self.getIdentifier(n.edge[0]), "\", { label : \"0\" });\n")
	s += fmt.Sprint("g.addEdge(\"", ownId, "\", \"", self.getIdentifier(n.edge[1]), "\", { label : \"1\" });\n")
	s += self.StringRecursive(n.edge[0])
	s += self.StringRecursive(n.edge[1])
	return s
}

func (self *RobddBuilder) String() string {
	return self.StringRecursive(self.bdd)
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
	bdd *Node
}

func (self *RobddBuilder) build(node *Element) *Node {
	self.output = node
	self.inputs = self.output.getAllInputs()
	self.inputsSize = len(self.inputs)
	self.bddTrue.id = 1
	self.bddFalse.id = 0

	self.bdd = self.buildRecursive(2)
	return self.bdd
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
		return self.mk(i, low, high)
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
	model.outputs[0].print()

	fmt.Println()
	fmt.Println()
	bdd := new(RobddBuilder)
	n := bdd.build(model.outputs[0])
	n.PrintTree()

	fmt.Println(bdd)
}