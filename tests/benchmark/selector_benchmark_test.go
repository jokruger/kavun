package benchmark

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

type selectorID uint8

const (
	selUnknown selectorID = iota
	selLen
	selType
	selName
	selFilter
	selEach
	selClear
	selAll
)

func dispatchByString(name string) int {
	switch name {
	case "all":
		return 7
	case "clear":
		return 6
	case "each":
		return 5
	case "len":
		return 1
	case "type":
		return 2
	case "name":
		return 3
	case "filter":
		return 4
	default:
		return 0
	}
}

func dispatchByID(id selectorID) int {
	switch id {
	case selAll:
		return 7
	case selClear:
		return 6
	case selEach:
		return 5
	case selLen:
		return 1
	case selType:
		return 2
	case selName:
		return 3
	case selFilter:
		return 4
	default:
		return 0
	}
}

func benchmarkStringDispatch(b *testing.B, names []string) {
	acc := 0
	l := len(names)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		acc += dispatchByString(names[i%l])
	}
	_ = acc
}

func benchmarkIDDispatch(b *testing.B, ids []selectorID) {
	acc := 0
	l := len(ids)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		acc += dispatchByID(ids[i%l])
	}
	_ = acc
}

func loadSelectorBenchmarkData(path string) ([]string, []selectorID, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 2 {
		return nil, nil, strconv.ErrSyntax
	}

	names := strings.Split(strings.TrimSpace(lines[0]), ", ")
	idParts := strings.Split(strings.TrimSpace(lines[1]), ", ")

	ids := make([]selectorID, 0, len(idParts))
	for _, part := range idParts {
		idVal, parseErr := strconv.Atoi(part)
		if parseErr != nil {
			return nil, nil, parseErr
		}
		ids = append(ids, selectorID(idVal))
	}

	if len(names) == 0 || len(ids) == 0 || len(names) != len(ids) {
		return nil, nil, strconv.ErrSyntax
	}

	return names, ids, nil
}

func BenchmarkSelector(b *testing.B) {
	names, ids, err := loadSelectorBenchmarkData("selector_test.lst")
	if err != nil {
		b.Fatalf("load selector_test.lst: %v", err)
	}

	b.Run("StringSwitch", func(b *testing.B) {
		benchmarkStringDispatch(b, names)
	})
	b.Run("IntSwitch", func(b *testing.B) {
		benchmarkIDDispatch(b, ids)
	})
}
