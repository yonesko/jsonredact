package jsonredact

import (
	"bytes"
	"reflect"
)

type selectorForest map[string]selectorForest

func (forest selectorForest) mergeWith(other selectorForest) {
	for k := range other {
		if forest[k] == nil {
			forest[k] = other[k]
		} else if !forest[k].hasChildren() {
			continue
		} else if !other[k].hasChildren() {
			clear(forest[k])
		} else {
			if reflect.ValueOf(forest).Pointer() == reflect.ValueOf(forest[k]).Pointer() &&
				reflect.ValueOf(other).Pointer() == reflect.ValueOf(other[k]).Pointer() {
				continue
			}
			forest[k].mergeWith(other[k])
		}
	}
}

func (forest selectorForest) hasChildren() bool {
	return len(forest) != 0
}

func (forest selectorForest) isTerminalMatch(key string) bool {
	return (forest[key] != nil && len(forest[key]) == 0) ||
		(forest["#"] != nil && len(forest["#"]) == 0) ||
		((key == "*" || key == "#") && forest[`\`+key] != nil && len(forest[`\`+key]) == 0)
}

func (forest selectorForest) add(str string) {
	var fi = forest
	elems := []string{}
	for i := 0; i < len(elems); i++ {
		if elems[i] == "*" {
			fi[elems[i+1]] = map[string]selectorForest{}
			fi["#"] = fi
			continue
		}
		if fi[elems[i]] == nil {
			fi[elems[i]] = map[string]selectorForest{}
		}
		fi = fi[elems[i]]
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
