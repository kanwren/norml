// Key comparison algorithm from https://github.com/yaml/go-yaml/blob/v3.0.4/sorter.go

package normalizer

import (
	"reflect"
	"sort"
	"unicode"

	"go.yaml.in/yaml/v3"
)

func sortMapKeys(content []*yaml.Node) []*yaml.Node {
	entries := len(content) / 2

	var keys mapKeys
	for i := range entries {
		n := content[i*2]
		var key mapKey
		key.index = i
		var value any
		n.Decode(&value)
		key.value = reflect.ValueOf(value)
		keys = append(keys, key)
	}
	sort.Stable(keys)

	newContent := make([]*yaml.Node, len(content))
	for i := range entries {
		newContent[i*2] = content[keys[i].index*2]
		newContent[i*2+1] = content[keys[i].index*2+1]
	}
	return newContent
}

type mapKeys []mapKey

type mapKey struct {
	index int
	value reflect.Value
}

func (m mapKeys) Len() int {
	return len(m)
}

func (m mapKeys) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

func deref(v reflect.Value) reflect.Value {
	for vk := v.Kind(); (vk == reflect.Interface || vk == reflect.Ptr) && !v.IsNil(); vk = v.Kind() {
		v = v.Elem()
	}
	return v
}

func (m mapKeys) Less(i, j int) bool {
	a, b := deref(m[i].value), deref(m[j].value)
	ak, bk := a.Kind(), b.Kind()

	aNum, aIsNumber := num(a)
	bNum, bIsNumber := num(b)
	if aIsNumber && bIsNumber {
		if aNum != bNum {
			return aNum < bNum
		}
		if ak != bk {
			return ak < bk
		}
		return numLess(a, b)
	}
	if ak != reflect.String || bk != reflect.String {
		return ak < bk
	}
	return stringNaturalLess([]rune(a.String()), []rune(b.String()))
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

func numLess(a, b reflect.Value) bool {
	switch a.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return a.Int() < b.Int()
	case reflect.Float32, reflect.Float64:
		return a.Float() < b.Float()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return a.Uint() < b.Uint()
	case reflect.Bool:
		return !a.Bool() && b.Bool()
	default:
		panic("not a number")
	}
}

func stringNaturalLess(a, b []rune) bool {
	digits := false
	i := 0
	for ; i < len(a) && i < len(b) && a[i] == b[i]; i++ {
		digits = unicode.IsDigit(a[i])
	}

	if i >= len(a) || i >= len(b) {
		return len(a) < len(b)
	}

	al := unicode.IsLetter(a[i])
	bl := unicode.IsLetter(b[i])
	if al && bl {
		return a[i] < b[i]
	}
	if al || bl {
		if digits {
			return al
		} else {
			return bl
		}
	}

	var ai, bi int
	var an, bn int64
	if a[i] == '0' || b[i] == '0' {
		for j := i - 1; j >= 0 && unicode.IsDigit(a[j]); j-- {
			if a[j] != '0' {
				an = 1
				bn = 1
				break
			}
		}
	}

	for ai = i; ai < len(a) && unicode.IsDigit(a[ai]); ai++ {
		an = an*10 + int64(a[ai]-'0')
	}
	for bi = i; bi < len(b) && unicode.IsDigit(b[bi]); bi++ {
		bn = bn*10 + int64(b[bi]-'0')
	}
	if an != bn {
		return an < bn
	}
	if ai != bi {
		return ai < bi
	}

	return a[i] < b[i]
}
