package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"unicode"
	"hash/fnv"
)

type ElementType int
const (
	NONE ElementType = iota
	INPUT
	OUTPUT
	AND
	OR
	NOT
	NAND
	NOR
	XOR
)

type Element struct {
	name string
	t ElementType
	inputs [2]*Element
}

const ELEMENTHASH_SIZE = 10

type ElementHash struct {
	el *Element
	next *ElementHash
}

type ElementsHash struct {
	elements [ELEMENTHASH_SIZE]*ElementHash
}

func hash(s string) int {
	h := fnv.New32a()
	h.Write([]byte(s))
	return int(h.Sum32()) % ELEMENTHASH_SIZE
}

func (h *ElementsHash) get(s string) *Element {
	n := h.elements[hash(s)]
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
	index := hash(t.name)
	elHashEntry := &ElementHash{t, nil}
	if h.elements[index] == nil {
		h.elements[index] = elHashEntry
	} else {
		hashEl := h.elements[index]
		for hashEl.next != nil {
			if hashEl.el.name == t.name {
				return
			}
			hashEl = hashEl.next
		}
		hashEl.next = elHashEntry
	}
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

func getElementTypeForTypeName(str string) ElementType {
	switch str {
	case "and": return AND
	case "or": return OR
	case "not": return NOT
	case "nand": return NAND
	case "nor": return NOR
	case "xor": return XOR
	}
	return NONE
}

func (self *Element) print() {
	fmt.Print("(")
	if self.inputs[0] != nil {
		self.inputs[0].print()
	}
	fmt.Print(")")
	fmt.Print(self.name, " ", self.t)
	fmt.Print("(")
	if self.inputs[1] != nil {
		self.inputs[1].print()
	}
	fmt.Print(")")
}

func main() {
	b, err := ioutil.ReadFile(os.Args[1]) // just pass the file name
	if err != nil {
		fmt.Print(err)
	}

	currentParsToken := RemoveWhitespaces(string(b))

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
		el := &Element{inputName, INPUT, [2]*Element{}}
		model.hash.add(el)
		model.inputs[i] = el
	}

	model.outputs = make([]*Element, len(rawOutputs))
	for i, outputName := range rawOutputs {
		el := &Element{outputName, OUTPUT, [2]*Element{}}
		model.hash.add(el)
		model.outputs[i] = el
	}

	for _, nodeData := range rawElements {
		elData := strings.FieldsFunc(nodeData, SplitElement)
		nodeName := elData[0]
		nodeType := elData[1]
		nodeInputs := make([]string, 2)
		nodeInputs[0] = elData[2]
		nodeInputs[1] = elData[3]

		elInputs := [2]*Element{}
		for j := 0; j < 2; j++ {
			elInputs[j] = model.hash.get(nodeInputs[j])
			if elInputs[j] == nil {
				elInputs[j] = &Element{nodeInputs[j], NONE, [2]*Element{}}
				model.hash.add(elInputs[j])
			}
		}

		el := model.hash.get(nodeName)
		if el == nil {
			el = &Element{nodeName, NONE, [2]*Element{}}
			model.hash.add(el)
		}
		el.inputs = elInputs
		el.t = getElementTypeForTypeName(nodeType)
	}

	model.outputs[0].print()
	/*currentParsToken = strings.SplitAfter(currentParsToken, "INPUT\n\t")[1]
	splitted := strings.Split(currentParsToken, ";")
	//inputs := splitted[0]
	currentParsToken = splitted[1]
	fmt.Println(currentParsToken)*/
}
