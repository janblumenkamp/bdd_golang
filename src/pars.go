package main

import (
	"fmt"
	"strings"
	"unicode"
	"hash/fnv"
)

type ElementType int
const (
	IN ElementType = iota
	OUT
	NOT
	AND
	OR
	NAND
	NOR
	XOR
)

type Element struct {
	name string
	elType ElementType
	val bool
	inputs []*Element
}

const ELEMENTHASH_SIZE = 10000

type ElementHash struct {
	el *Element
	next *ElementHash
}

type ElementsHash struct {
	elements [ELEMENTHASH_SIZE]*ElementHash
	amount int
}

func hashElement(s string) int {
	h := fnv.New32a()
	h.Write([]byte(s))
	return int(h.Sum32()) % ELEMENTHASH_SIZE
}

func (h *ElementsHash) get(s string) *Element {
	n := h.elements[hashElement(s)]
	if n == nil {
		return nil
	}
	for n != nil && n.el.name != s {
		n = n.next
	}
	if n == nil {
		return nil
	}
	return n.el
}

func (h *ElementsHash) add(t *Element) {
	index := hashElement(t.name)
	elHashEntry := &ElementHash{t, nil}
	if h.elements[index] == nil {
		h.elements[index] = elHashEntry
	} else {
		hashEl := h.elements[index]
		for hashEl.next != nil {
			hashEl = hashEl.next
		}
		hashEl.next = elHashEntry
	}
	h.amount ++
}

func deepCopy(element *Element) *Element {
	if element == nil {
		return nil
	}

	copyEl := new(Element)
	copyEl.name = element.name
	copyEl.val = element.val
	copyEl.elType = element.elType
	copyEl.inputs = make([]*Element, len(element.inputs))

	for i, input := range element.inputs {
		copyEl.inputs[i] = deepCopy(input)
	}
	return copyEl
}

func (self *Element) evalInputs(op func(bool, bool) bool) bool {
	val := self.inputs[0].eval()
	for i := 1; i < len(self.inputs); i++ {
		val = op(val, self.inputs[i].eval())
	}
	return val
}

func (self *Element) eval() bool {
	switch self.elType {
	case IN:
	case OUT: 	self.val = self.inputs[0].eval(); break
	case NOT: 	self.val = !self.inputs[0].eval(); break
	case AND:	self.val = self.evalInputs(func(a bool, b bool) bool { return a && b }); break
	case OR: 	self.val = self.evalInputs(func(a bool, b bool) bool { return a || b }); break
	case NAND:	self.val = self.evalInputs(func(a bool, b bool) bool { return !(a && b) }); break
	case NOR: 	self.val = self.evalInputs(func(a bool, b bool) bool { return !(a || b) }); break
	case XOR: 	self.val = self.evalInputs(func(a bool, b bool) bool { return a != b }); break
	}
	return self.val
}

type Model struct {
	hash ElementsHash
	name string
	inputs []*Element
	outputs []*Element
}

// https://stackoverflow.com/questions/32081808/strip-all-whitespace-from-a-string
func RemoveWhitespaces(str string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, str)
}

func SplitElement(r rune) bool {
	return r == '=' || r == ',' || r == '(' || r == ')'
}

func (self *Element) printRecursive(intendation int) {
	for i := 0; i < intendation; i++ {
		fmt.Print("  ")
	}
	fmt.Print(self.name, " ", self.elType, "(", self.val, ")")
	for _, el := range self.inputs {
		fmt.Println()
		if el != nil {
			el.printRecursive(intendation + 1)
		}
	}
}

func (self *Element) print() {
	self.printRecursive(0)
	fmt.Println()
}

func (self *Element) collectAllInputs(hash *ElementsHash) {
	if self.elType == IN && hash.get(self.name) == nil {
		hash.add(self)
	}
	for _, el := range self.inputs {
		el.collectAllInputs(hash)
	}
}

func (self *Model) getAllInputs(el *Element) []*Element {
	hash := ElementsHash{}
	el.collectAllInputs(&hash)
	elements := make([]*Element, hash.amount)
	currentElementsIndex := 0
	for _, inp := range self.inputs {
		if hash.get(inp.name) != nil {
			elements[currentElementsIndex] = inp
			currentElementsIndex ++
		}
	}

	return elements
}

func pars(s string) *Model {
	currentParsToken := RemoveWhitespaces(s)

	currentParsToken = strings.Split(currentParsToken, "MODULE")[1]
	model := Model{}

	split := strings.Split(currentParsToken, "INPUT")
	currentParsToken = split[1]
	model.name = split[0]

	split = strings.Split(currentParsToken, ";OUTPUT")
	currentParsToken = split[1]
	rawInputs := strings.Split(split[0], ",")

	split = strings.Split(currentParsToken, ";STRUCTURE")
	currentParsToken = split[1]
	rawOutputs := strings.Split(split[0], ",")

	split = strings.Split(currentParsToken, ";END") // Should be ENDMODULE, but MODULE was splitted in first step
	rawElements := strings.Split(split[0], ";")

	model.inputs = make([]*Element, len(rawInputs))
	for i, inputName := range rawInputs {
		el := &Element{inputName, IN, false, nil}
		model.hash.add(el)
		model.inputs[i] = el
	}

	model.outputs = make([]*Element, len(rawOutputs))
	for i, outputName := range rawOutputs {
		el := &Element{outputName, OUT, false, nil}
		model.hash.add(el)
		model.outputs[i] = el
	}

	for _, nodeData := range rawElements {
		elData := strings.FieldsFunc(nodeData, SplitElement)
		nodeName := elData[0]
		nodeType := OUT
		elInputs := []*Element{}
		if len(elData) > 2 {
			switch elData[1] {
			case "not": 	nodeType = NOT; break
			case "and": 	nodeType = AND; break
			case "or": 		nodeType = OR; break
			case "nand": 	nodeType = NAND; break
			case "nor": 	nodeType = NOR; break
			case "xor": 	nodeType = XOR; break
			}

			inputAmount := len(elData) - 2
			elInputs = make([]*Element, inputAmount)
			for i := 0; i < inputAmount; i++ {
				currentInputName := elData[i + 2]
				elInputs[i] = model.hash.get(currentInputName)
				if elInputs[i] == nil {
					elInputs[i] = &Element{currentInputName, IN, false, nil}
					model.hash.add(elInputs[i])
				}
			}
		} else {
			elInputs = make([]*Element, 1)
			elInputs[0] = model.hash.get(elData[1])
			if elInputs[0] == nil {
				elInputs[0] = &Element{elData[1], IN, false, nil}
				model.hash.add(elInputs[0])
			}
		}

		el := model.hash.get(nodeName)
		if el == nil {
			el = &Element{nodeName, IN, false, nil}
			model.hash.add(el)
		}
		el.inputs = elInputs
		el.elType = nodeType
	}

	return &model
}