package main

import (
	"fmt"
	"io/ioutil"
	//"os"
	"time"
	"flag"
	"os"
	"log"
	"runtime/pprof"
)

import _ "net/http/pprof"


type Node struct {
	variable int
	id int
	edge [2]*Node
}

var nodeID = 0
func createNode(variable int, edge0 *Node, edge1 *Node) *Node  {
	n := new(Node)
	n.id = nodeID
	nodeID ++
	n.variable = variable
	n.edge = [2]*Node{edge0, edge1}
	return n
}

const NODEHASH_SIZE = 997

func pair(i int, j int) int {
	return (((i + j) * (i + j + 1)) / 2) + i
}

func (h *NodesHash) hashIndex(i int, low *Node, high *Node) int {
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

type NodesCache struct {
	elements [10000]*Node
}

func (h *NodesCache) hashIndex(low *Node, high *Node) int {
	index := 0
	if low != nil && high != nil {
		index = pair(low.id, high.id)
	}
	return index % NODEHASH_SIZE
}

func (self *NodesCache) add(node *Node) {
	self.elements[self.hashIndex(node.edge[0], node.edge[1])] = node
}

func (self *NodesCache) get(low *Node, high *Node) *Node {
	return self.elements[self.hashIndex(low, high)]
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

var numberOfCollisions = 0
func (h *NodesHash) add(t *Node) {
	elNodeHashEntry := &NodeHash{t, nil}
	index := h.hashIndex(t.id, t.edge[0], t.edge[1])
	if h.elements[index] == nil {
		h.elements[index] = elNodeHashEntry
	} else {
		hashEL := h.elements[index]
		for hashEL.next != nil {
			numberOfCollisions ++
			hashEL = hashEL.next
		}
		hashEL.next = elNodeHashEntry
	}
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
	bddTrue *Node
	bddFalse *Node
	bdd *Node
}

func (self *Node) isFinal() bool {
	return self.id == 0 || self.id == 1
}

func (self *RobddBuilder) build(model *Model, node *Element) *Node {
	self.output = node
	self.inputs = model.getAllInputs(self.output)
	self.inputsSize = len(self.inputs)
	fmt.Println(self.inputsSize, " inputs")
	self.bdd = self.buildRecursive(2)
	return self.bdd
}

func (self *RobddBuilder) init() {
	self.bddFalse = createNode(0, nil, nil)
	self.bddTrue = createNode(1, nil, nil)
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
			return self.bddTrue
		} else {
			return self.bddFalse
		}
	} else {
		self.inputs[i - 2].val = false
		low := self.buildRecursive(i + 1)
		self.inputs[i - 2].val = true
		high := self.buildRecursive(i + 1)
		return self.mk(i, low, high)
	}
}

func (self *RobddBuilder) addVar(variable int) *Node {
	if variable > self.bddTrue.variable {
		self.bddTrue.variable = variable + 1
	}
	if variable > self.bddFalse.variable {
		self.bddFalse.variable = variable + 1
	}
	return self.mk(variable, self.bddFalse, self.bddTrue)
}

var cpuprofile = flag.String("cpuprofile", "./prof", "write cpu profile to `file`")

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	b, errIn := ioutil.ReadFile("/home/jan/Documents/Uni/WiSe17/TI1_Vertiefung/iscas85/iscas85/trace/c1.trace")//os.Args[1])
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
	errOut := ioutil.WriteFile("/home/jan/Documents/Uni/WiSe17/TI1_Vertiefung/logicmerger/graphviz/graph.js", d1, 0644)
	if errOut != nil {
		fmt.Print(errOut)
	}
}
