package jsonredact

import (
	"bytes"
	"github.com/tidwall/gjson"
)

type selectorForest map[string]selectorForest

func (forest selectorForest) selectForest(key gjson.Result) selectorForest {
	f, ok := forest[key.String()]
	if ok {
		return f
	}
	f, ok = forest[`\`+key.String()]
	if ok {
		return f
	}
	f, ok = forest["#"]
	if ok {
		return f
	}
	f, ok = forest["*"]
	if ok {
		f["*"] = f
		return f
	}
	return f
}

func (forest selectorForest) mergeWith(other selectorForest) {
	for k, v := range other {
		if forest[k] == nil {
			forest[k] = v
		} else if !forest[k].hasChildren() {
			continue
		} else if !other[k].hasChildren() {
			clear(forest[k])
		} else {
			forest[k].mergeWith(v)
		}
	}
}

func (forest selectorForest) hasChildren() bool {
	return len(forest) != 0
}

func (forest selectorForest) add(str string) {
	var fi = forest
	for _, val := range splitSelectorExpression(str) {
		if fi[val] == nil {
			fi[val] = map[string]selectorForest{}
		}
		fi = fi[val]
	}
}

func (forest selectorForest) string(prefix string) string {
	buffer := bytes.Buffer{}
	for k, v := range forest {
		if v.hasChildren() {
			buffer.WriteString(v.string(prefix + k + `->`))
		} else {
			buffer.WriteString(prefix + k + "\n")
		}
	}
	return buffer.String()
}

func (forest selectorForest) String() string {
	return forest.string("")
}

func (forest selectorForest) Clone() selectorForest {
	f := selectorForest{}
	for k, v := range forest {
		f[k] = v.Clone()
	}
	return f
}
