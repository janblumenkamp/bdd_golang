package main

import (
	"fmt"
	"io/ioutil"
	"sync"
	"time"
	"os"
	"strconv"

	_ "net/http/pprof"
	"runtime"
	"flag"
	"log"
	"runtime/pprof"
)

// One BDD Node consists of the variable order, the unique id and two edge pointers to the two
// child nodes
type Node struct {
	variable int
	id int
	edge [2]*Node
}

var nodeID = 0
// Create a new node with the given variable order and the two child nodes.
// The unique id is incremented automatically.
func createNode(variable int, edge0 *Node, edge1 *Node) *Node  {
	n := new(Node)
	n.id = nodeID
	nodeID ++
	n.variable = variable
	n.edge = [2]*Node{edge0, edge1}
	return n
}

const NODEHASH_SIZE = 14593
const NODECACHE_SIZE = 997

// Hash function as suggested in
// https://www.cs.utexas.edu/~isil/cs389L/bdd.pdf
func pair(i int, j int) int {
	return (((i + j) * (i + j + 1)) / 2) + i
}

// Hash function as suggested in
// https://www.cs.utexas.edu/~isil/cs389L/bdd.pdf
func (h *NodesHash) hashIndex(i int, low *Node, high *Node) int {
	index := i
	if low != nil && high != nil {
		index = pair(i, pair(low.id, high.id))
	}
	return index % NODEHASH_SIZE
}

// Node hash to check if a given Node already exists. Hash implemented as chained hash
// (chain in linked list in case of collision)
type NodeHash struct {
	el *Node
	next *NodeHash
}

// Nodeshash storage structure
type NodesHash struct {
	elements [NODEHASH_SIZE]*NodeHash
	numberOfCollisions int
}

// Cache for Nodes, important for apply function but not used at the moment
// due to performance reasons. It is necessary to create a cache for each
// logical operation
type NodesCache struct {
	elements [NODECACHE_SIZE]*Node
}

// Calculates the nodecache index for a given low and high node
func (h *NodesCache) hashIndex(low *Node, high *Node) int {
	index := 0
	if low != nil && high != nil {
		index = pair(low.variable, high.variable)
	}
	return index % NODECACHE_SIZE
}

// Add the given node to the cache
func (self *NodesCache) set(node *Node) {
	self.elements[self.hashIndex(node.edge[0], node.edge[1])] = node
}

// Get the node with the two childs from the cache
// Returns nil if not available
func (self *NodesCache) get(low *Node, high *Node) *Node {
	el := self.elements[self.hashIndex(low, high)]
	if el != nil && el.edge[0] == low && el.edge[1] == high {
		return el
	}
	return nil
}

// Get a node with the given variable ordering and the given low and high nodes
// from the nodes hash
func (h *NodesHash) get(variable int, low *Node, high *Node) *Node {
	n := h.elements[h.hashIndex(variable, low, high)]
	if n == nil {
		return nil
	}
	for n != nil && (n.el.variable != variable || n.el.edge[0] != low || n.el.edge[1] != high) {
		n = n.next
	}
	if n == nil {
		return nil
	}
	return n.el
}

// Adds the given node to the node hash without checking if it already exists
func (h *NodesHash) add(t *Node) {
	elNodeHashEntry := &NodeHash{t, nil}
	index := h.hashIndex(t.variable, t.edge[0], t.edge[1])
	if h.elements[index] == nil {
		h.elements[index] = elNodeHashEntry
	} else {
		hashEL := h.elements[index]
		for hashEL.next != nil {
			h.numberOfCollisions ++
			hashEL = hashEL.next
		}
		hashEL.next = elNodeHashEntry
	}
}

// converts the given node to a string for easier represenation
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

// Returns the unique node id as a string (in the terminal nodes true or false)
func (self *RobddBuilder) getIdentifier(node *Node) string {
	if node == nil {
		return ""
	}

	if node.id == 0 {
		return "false"
	} else if node.id == 1 {
		return "true"
	} else {
		return fmt.Sprint(self.inputs[node.variable - 1].name, "_", node.id)
	}
}

// generates the javascript bdd tree representation recursively for the given node
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

// generates the javascript bdd tree representation recursively for the base of the
// bdd builder. Adds the whole draw function in JS
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

// Checks if two nodes and it's subnodes (the whole bdd) equals the given
func (first *Node) equals(second *Node) bool {
	if first == nil && second == nil {
		return true
	} else if first != nil && second != nil {
		return first.edge[0].equals(second.edge[0]) && first.edge[1].equals(second.edge[1]) && first.id == second.id
	}
	return false
}

// Data structure to store the bdd builder
type RobddBuilder struct {
	creationHash NodesHash
	creationHashMutex sync.Mutex
	nodeCache NodesCache
	output *Element
	inputs []*Element
	inputsSize int
	bddTrue *Node
	bddFalse *Node
	bdd *Node
}

// Checks if the given node is a final node or not
func (self *Node) isFinal() bool {
	return self.id == 0 || self.id == 1
}

// Returns the node variable ordering of the given element
func (self *RobddBuilder) getVarForEl(element *Element) int {
	for i, el := range self.inputs {
		if el == element {
			return i + 1
		}
	}
	return 0
}

// Generic apply method for a bdd as mentioned in
// https://www.cs.utexas.edu/~isil/cs389L/bdd.pdf and
// https://software.intel.com/en-us/articles/multicore-enabling-a-binary-decision-diagram-algorithm
// Here the actual parallelization happens. The given logical operation is applied to the given nodes
// x and y and the resulting node is transferred through a channel to allow parallelization.
func (self *RobddBuilder) apply(op func(bool, bool) bool, x *Node, y *Node, result chan *Node) {
	u := new(Node)
	if x.isFinal() && y.isFinal() {
		if op(x.id == 1, y.id == 1) {
			u = self.bddTrue
		} else {
			u = self.bddFalse
		}
	} else {
		low := make(chan *Node, 1)
		high := make(chan *Node, 1)
		if x.variable == y.variable {
			go self.apply(op, x.edge[0], y.edge[0], low)
			self.apply(op, x.edge[1], y.edge[1], high)
			u = self.mk(x.variable, <-low, <-high)
		} else if x.variable < y.variable {
			go self.apply(op, x.edge[0], y, low)
			self.apply(op, x.edge[1], y, high)
			u = self.mk(x.variable, <-low, <-high)
		} else if x.variable > y.variable {
			go self.apply(op, x, y.edge[0], low)
			self.apply(op, x, y.edge[1], high)
			u = self.mk(y.variable, <-low, <-high)
		}
	}
	result <- u
}

// Apply a logical not to a bdd by inverting the leaves recursivley
func (self *RobddBuilder) applyNot(x *Node) *Node {
	if x.isFinal() {
		if x == self.bddTrue {
			return self.bddFalse
		} else {
			return self.bddTrue
		}
	}
	return self.mk(x.variable, self.applyNot(x.edge[0]), self.applyNot(x.edge[1]))
}

// top level build function that initializes the BDD and builds
// the bdd for the given element in the given model
func (self *RobddBuilder) build(model *Model, node *Element) *Node {
	self.bddFalse = createNode(0, nil, nil)
	self.bddTrue = createNode(1, nil, nil)
	self.output = node
	self.inputs = model.getAllInputs(self.output)
	self.inputsSize = len(self.inputs)
	self.bddTrue.variable = self.inputsSize + 1
	self.bddFalse.variable = self.inputsSize + 1

	fmt.Println(self.inputsSize, " inputs")
	self.bdd = self.buildRecursive(node)
	return self.bdd
}

// BDD mk function
// refer https://www.cs.utexas.edu/~isil/cs389L/bdd.pdf
// Creates a variable with the given order and low and high nodes if it does
// not already exist
func (self *RobddBuilder) mk(variable int, low *Node, high *Node) *Node {
	if low == high {
		return low
	}
	self.creationHashMutex.Lock()
	n := self.creationHash.get(variable, low, high)
	if n == nil {
		n = createNode(variable, low, high)
		self.creationHash.add(n)
	}
	self.creationHashMutex.Unlock()
	return n
}

// One element can have multiple inputs so the evaluation method is applied to each of the inputs recursively
func (self *RobddBuilder) buildRecursiveApplyToInputs(op func(bool, bool) bool, element *Element) *Node {
	node := self.buildRecursive(element.inputs[0])
	nodeResult := make(chan *Node, 1)
	for i := 1; i < len(element.inputs); i++ {
		self.apply(op, node, self.buildRecursive(element.inputs[i]), nodeResult)
		node = <-nodeResult
	}
	return node
}

// Builds the bdd based on the given element recursively
func (self *RobddBuilder) buildRecursive(element *Element) *Node {
	for _, el := range element.inputs {
		self.buildRecursive(el)
	}
	switch element.elType {
	case IN:    return self.addVar(self.getVarForEl(element))
	case OUT: 	return self.buildRecursive(element.inputs[0])
	case NOT: 	return self.applyNot(self.buildRecursive(element.inputs[0]))
	case AND:	return self.buildRecursiveApplyToInputs(func(a bool, b bool) bool { return a && b }, element)
	case OR: 	return self.buildRecursiveApplyToInputs(func(a bool, b bool) bool { return a || b }, element)
	case NAND:	return self.buildRecursiveApplyToInputs(func(a bool, b bool) bool { return !(a && b) }, element)
	case NOR: 	return self.buildRecursiveApplyToInputs(func(a bool, b bool) bool { return !(a || b) }, element)
	case XOR: 	return self.buildRecursiveApplyToInputs(func(a bool, b bool) bool { return a != b }, element)
	}
	return nil
}

// Adds a new variable with the given variable order to the BDD
func (self *RobddBuilder) addVar(variable int) *Node {
	if variable >= self.bddTrue.variable {
		self.bddTrue.variable = variable + 1
	}
	if variable >= self.bddFalse.variable {
		self.bddFalse.variable = variable + 1
	}
	return self.mk(variable, self.bddFalse, self.bddTrue)
}

// Actual processing of the bdd. Reads the trace file, parses it, generates the bdd based
// on the selected input and writes it to the JS file
func proc() {
	b, errIn := ioutil.ReadFile(os.Args[1])
	if errIn != nil {
		fmt.Print(errIn)
	}

	model := new(Model)

	start := time.Now()
	model = pars(string(b))
	fmt.Println(time.Since(start))

	for i, el := range model.outputs {
		fmt.Println(i, ": ", len(model.getAllInputs(el)))
	}


	fmt.Println()
	fmt.Println()
	bdd := new(RobddBuilder)
	start = time.Now()
	index, err := strconv.ParseInt(os.Args[2], 10, 16)
	if err != nil {
	  fmt.Println("error argument")
	  return
	}
	bdd.build(model, model.outputs[index])
	fmt.Println("built in ", time.Since(start))
	fmt.Println("number of collisions:", bdd.creationHash.numberOfCollisions)

	d1 := []byte(bdd.String())
	errOut := ioutil.WriteFile(os.Args[3], d1, 0644)
	if errOut != nil {
		fmt.Print(errOut)
	}
}

var cpuprofile = flag.String("cpuprofile", "./prof", "write cpu profile to `file`")

// Main function setting up the profiler and calling the actual processing function.
// Three arguments are taken:
// 1) The path to the input file
// 2) The index of the output to pars
// 3) The path to the js file to write the BDD to
// example:
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
	runtime.GOMAXPROCS(8)
	fmt.Println("Max num of cores: ", runtime.NumCPU())
	proc()
}
