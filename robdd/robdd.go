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

const NODEHASH_SIZE = 15485863

func (h *NodesHash) hashIndex(i int, low *Node, high *Node) int {
	pair := func(i int, j int) int {
		return (((i + j) * (i + j + 1)) / 2) + i
	}

	index := i
	if low != nil && high != nil {
		index = pair(i, pair(low.id, high.id))
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

func (h *NodesHash) get(i int, low *Node, high *Node) *Node {
	n := h.elements[h.hashIndex(i, low, high)]
	if n == nil {
		return nil
	}
	for n != nil && (n.el.id != i || n.el.edge[0] != low || n.el.edge[1] != high) {
		n = n.next
	}
	if n == nil {
		return nil
	}
	return n.el
}

func (h *NodesHash) add(t *Node) {
	elNodeHashEntry := &NodeHash{t, nil}
	index := h.hashIndex(t.id, t.edge[0], t.edge[1])
	if h.elements[index] == nil {
		h.elements[index] = elNodeHashEntry
	} else {
		hashEL := h.elements[index]
		for hashEL.next != nil {
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
	s := `window.onload = function() {
	var g = new Graph();
	Math.seedrandom("bddseed");
	g.edgeFactory.template.style.directed = true;`
	s += self.StringRecursive(self.bdd)
	s += `var layouter = new Graph.Layout.Ordered(g, topological_sort(g));
	layouter.layout();
	var renderer = new Graph.Renderer.Raphael('canvas', g, 1000, 800);
	renderer.draw();
	};`
	return s
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

func (self *RobddBuilder) build(model *Model, node *Element) *Node {
	self.output = node
	self.inputs = model.getAllInputs(self.output)
	self.inputsSize = len(self.inputs)
	fmt.Println(self.inputsSize, " inputs")
	self.bddTrue.id = 1
	self.bddFalse.id = 0

	self.bdd = self.buildRecursive(2)
	return self.bdd
}

func (self *RobddBuilder) mk(id int, low *Node, high *Node) *Node {
	if low == high {
		return low
	}
	n := self.creationHash.get(id, low, high)
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
	b, errIn := ioutil.ReadFile(os.Args[1]) // just pass the file name
	if errIn != nil {
		fmt.Print(errIn)
	}

	model := new(Model)

	start := time.Now()
	model = pars(string(b))
	fmt.Println(time.Since(start))
	//model.outputs[0].print()

	for i, el := range model.outputs {
		fmt.Println(i, ": ", len(model.getAllInputs(el)))
	}

//	model.outputs[0].print()


	fmt.Println()
	fmt.Println()
	bdd := new(RobddBuilder)
	start = time.Now()
	bdd.build(model, model.outputs[0])
	fmt.Println("built in ", time.Since(start))

	fmt.Println("number of collisions:", numberOfCollisions)

	d1 := []byte(bdd.String())
	errOut := ioutil.WriteFile(os.Args[2], d1, 0644)
	if errOut != nil {
		fmt.Print(errOut)
	}
}