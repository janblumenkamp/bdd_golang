package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"unicode"
	"hash/fnv"
)

type Element struct {
	name string
	elType string
	inputs []*Element
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

func (self *Element) print(intendation int) {
	for i := 0; i < 2 * intendation; i++ {
		fmt.Print(" ")
	}
	fmt.Print(self.name, " ", self.elType)
	for _, el := range self.inputs {
		fmt.Println()
		if el != nil {
			el.print(intendation + 1)
		}
	}
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
		el := &Element{inputName, "in", nil}
		model.hash.add(el)
		model.inputs[i] = el
	}

	model.outputs = make([]*Element, len(rawOutputs))
	for i, outputName := range rawOutputs {
		el := &Element{outputName, "out", nil}
		model.hash.add(el)
		model.outputs[i] = el
	}

	for _, nodeData := range rawElements {
		elData := strings.FieldsFunc(nodeData, SplitElement)
		nodeName := elData[0]
		nodeType := elData[1]

		inputAmount := len(elData) - 2
		elInputs := make([]*Element, inputAmount)
		for i := 0; i < inputAmount; i++ {
			currentInputName := elData[i + 2]
			elInputs[i] = model.hash.get(currentInputName)
			if elInputs[i] == nil {
				elInputs[i] = &Element{currentInputName, "", nil}
				model.hash.add(elInputs[i])
			}
		}

		el := model.hash.get(nodeName)
		if el == nil {
			el = &Element{nodeName, "", nil}
			model.hash.add(el)
		}
		el.inputs = elInputs
		el.elType = nodeType
	}

	model.outputs[1].print(0)
}
