// Key comparison algorithm adapted from https://github.com/yaml/go-yaml/blob/v3.0.4/sorter.go

package normalizer

import (
	"cmp"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"unicode"

	"go.yaml.in/yaml/v3"
)

func sortMapKeys(content []*yaml.Node) ([]*yaml.Node, error) {
	entries := len(content) / 2
	if entries == 0 {
		return content, nil
	}

	// Check if all keys are strings (the overwhelmingly common case).
	// Non-string keys (int, bool, null, float, complex) use the mixed path.
	allStrings := true
	for i := 0; i < entries; i++ {
		n := content[i*2]
		if n.Kind != yaml.ScalarNode || n.Tag != "!!str" {
			allStrings = false
			break
		}
	}

	if allStrings {
		return sortStringKeys(content, entries)
	}
	return sortMixedKeys(content, entries)
}

// sortStringKeys sorts string-keyed maps in-place, avoiding allocations.
func sortStringKeys(content []*yaml.Node, entries int) ([]*yaml.Node, error) {
	// Check if already sorted
	sorted := true
	for i := 1; i < entries; i++ {
		if stringNaturalCmp(content[(i-1)*2].Value, content[i*2].Value) > 0 {
			sorted = false
			break
		}
	}
	if sorted {
		return content, nil
	}

	// Sort in-place using sort.Interface to swap key-value pairs together
	sort.Stable(stringKeyPairs(content))
	return content, nil
}

// stringKeyPairs wraps a content slice to sort key-value pairs in-place.
type stringKeyPairs []*yaml.Node

func (s stringKeyPairs) Len() int { return len(s) / 2 }

func (s stringKeyPairs) Swap(i, j int) {
	// Swap both key and value together
	s[i*2], s[j*2] = s[j*2], s[i*2]
	s[i*2+1], s[j*2+1] = s[j*2+1], s[i*2+1]
}

func (s stringKeyPairs) Less(i, j int) bool {
	return stringNaturalCmp(s[i*2].Value, s[j*2].Value) < 0
}

// keyKind represents the type of a map key for sorting purposes.
type keyKind int

const (
	keyKindNull keyKind = iota
	keyKindBool
	keyKindInt
	keyKindFloat
	keyKindString
	keyKindOther // Complex keys (maps, sequences) - rare
)

// mixedKey includes complexVal for non-scalar keys.
type mixedKey struct {
	index      int
	kind       keyKind
	intVal     int64
	floatVal   float64
	strVal     string
	complexVal reflect.Value
}

// sortMixedKeys handles maps with non-scalar keys (rare).
func sortMixedKeys(content []*yaml.Node, entries int) ([]*yaml.Node, error) {
	keys := make([]mixedKey, entries)
	for i := range entries {
		key, err := makeMixedKey(i, content[i*2])
		if err != nil {
			return nil, err
		}
		keys[i] = key
	}

	if slices.IsSortedFunc(keys, mixedKeyCmp) {
		return content, nil
	}

	slices.SortStableFunc(keys, mixedKeyCmp)

	newContent := make([]*yaml.Node, len(content))
	for i := range entries {
		newContent[i*2] = content[keys[i].index*2]
		newContent[i*2+1] = content[keys[i].index*2+1]
	}
	return newContent, nil
}

func makeMixedKey(index int, n *yaml.Node) (mixedKey, error) {
	key := mixedKey{index: index}

	if n.Kind != yaml.ScalarNode {
		key.kind = keyKindOther
		var value any
		if err := n.Decode(&value); err != nil {
			return mixedKey{}, err
		}
		key.complexVal = reflect.ValueOf(value)
		return key, nil
	}

	switch n.Tag {
	case "!!null":
		key.kind = keyKindNull
	case "!!bool":
		key.kind = keyKindBool
		if n.Value == "true" {
			key.intVal = 1
		}
	case "!!int":
		if v, err := strconv.ParseInt(n.Value, 0, 64); err == nil {
			key.kind = keyKindInt
			key.intVal = v
		} else {
			key.kind = keyKindString
			key.strVal = n.Value
		}
	case "!!float":
		if v, err := strconv.ParseFloat(n.Value, 64); err == nil {
			key.kind = keyKindFloat
			key.floatVal = v
		} else {
			key.kind = keyKindString
			key.strVal = n.Value
		}
	default:
		key.kind = keyKindString
		key.strVal = n.Value
	}

	return key, nil
}

func mixedKeyCmp(a, b mixedKey) int {
	if a.kind != b.kind {
		return cmp.Compare(a.kind, b.kind)
	}

	switch a.kind {
	case keyKindNull:
		return 0
	case keyKindBool, keyKindInt:
		return cmp.Compare(a.intVal, b.intVal)
	case keyKindFloat:
		return cmp.Compare(a.floatVal, b.floatVal)
	case keyKindString:
		return stringNaturalCmp(a.strVal, b.strVal)
	case keyKindOther:
		return complexCmp(a.complexVal, b.complexVal)
	}
	return 0
}

// complexCmp compares complex keys using reflection (rare case)
func complexCmp(a, b reflect.Value) int {
	a, b = deref(a), deref(b)
	ak, bk := a.Kind(), b.Kind()

	aNum, aIsNumber := num(a)
	bNum, bIsNumber := num(b)
	if aIsNumber && bIsNumber {
		if aNum != bNum {
			return cmp.Compare(aNum, bNum)
		}
		if ak != bk {
			return cmp.Compare(ak, bk)
		}
		return numCmp(a, b)
	}
	if ak != reflect.String || bk != reflect.String {
		return cmp.Compare(ak, bk)
	}
	return stringNaturalCmp(a.String(), b.String())
}

func deref(v reflect.Value) reflect.Value {
	for vk := v.Kind(); (vk == reflect.Interface || vk == reflect.Ptr) && !v.IsNil(); vk = v.Kind() {
		v = v.Elem()
	}
	return v
}

func num(v reflect.Value) (f float64, ok bool) {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(v.Int()), true
	case reflect.Float32, reflect.Float64:
		return v.Float(), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return float64(v.Uint()), true
	case reflect.Bool:
		if v.Bool() {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

func numCmp(a, b reflect.Value) int {
	switch a.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return cmp.Compare(a.Int(), b.Int())
	case reflect.Float32, reflect.Float64:
		return cmp.Compare(a.Float(), b.Float())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return cmp.Compare(a.Uint(), b.Uint())
	case reflect.Bool:
		if a.Bool() == b.Bool() {
			return 0
		}
		if !a.Bool() {
			return -1
		}
		return 1
	default:
		panic("not a number")
	}
}

// stringNaturalCmp compares strings with natural number ordering, returning -1, 0, or 1.
// For example: "a2" < "a10" (because 2 < 10 numerically)
func stringNaturalCmp(a, b string) int {
	ar, br := []rune(a), []rune(b)

	digits := false
	i := 0
	for ; i < len(ar) && i < len(br) && ar[i] == br[i]; i++ {
		digits = unicode.IsDigit(ar[i])
	}

	if i >= len(ar) || i >= len(br) {
		return cmp.Compare(len(ar), len(br))
	}

	al := unicode.IsLetter(ar[i])
	bl := unicode.IsLetter(br[i])
	if al && bl {
		return cmp.Compare(ar[i], br[i])
	}
	if al || bl {
		if digits {
			if al {
				return -1
			}
			return 1
		}
		if bl {
			return -1
		}
		return 1
	}

	var ai, bi int
	var an, bn int64
	if ar[i] == '0' || br[i] == '0' {
		for j := i - 1; j >= 0 && unicode.IsDigit(ar[j]); j-- {
			if ar[j] != '0' {
				an = 1
				bn = 1
				break
			}
		}
	}

	for ai = i; ai < len(ar) && unicode.IsDigit(ar[ai]); ai++ {
		an = an*10 + int64(ar[ai]-'0')
	}
	for bi = i; bi < len(br) && unicode.IsDigit(br[bi]); bi++ {
		bn = bn*10 + int64(br[bi]-'0')
	}
	if an != bn {
		return cmp.Compare(an, bn)
	}
	if ai != bi {
		return cmp.Compare(ai, bi)
	}

	return cmp.Compare(ar[i], br[i])
}
