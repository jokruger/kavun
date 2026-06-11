package compiler

import (
	"fmt"
	"slices"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/core"
)

type StaticBuilder struct {
	static            core.Static
	primitives        map[core.Value]int
	decimals          map[string]int
	strings           map[string]int
	runes             map[string]int
	formatSpecs       map[string]int
	compiledFunctions map[string]int
}

func NewStaticBuilder() *StaticBuilder {
	s := core.Static{
		Primitives:        make([]core.Value, 0),
		Decimals:          make([]dec128.Dec128, 0),
		Strings:           make([]string, 0),
		Runes:             make([]core.Runes, 0),
		FormatSpecs:       make([]core.FormatSpec, 0),
		CompiledFunctions: make([]core.CompiledFunction, 0),
	}

	return &StaticBuilder{
		static:            s,
		primitives:        make(map[core.Value]int),
		decimals:          make(map[string]int),
		strings:           make(map[string]int),
		runes:             make(map[string]int),
		formatSpecs:       make(map[string]int),
		compiledFunctions: make(map[string]int),
	}
}

func (b *StaticBuilder) Build() core.Static {
	return core.Static{
		Primitives:        slices.Clip(b.static.Primitives),
		Decimals:          slices.Clip(b.static.Decimals),
		Strings:           slices.Clip(b.static.Strings),
		Runes:             slices.Clip(b.static.Runes),
		FormatSpecs:       slices.Clip(b.static.FormatSpecs),
		CompiledFunctions: slices.Clip(b.static.CompiledFunctions),
	}
}

func (b *StaticBuilder) AddPrimitive(v core.Value) int {
	if i, ok := b.primitives[v]; ok {
		return i
	}
	i := len(b.static.Primitives)
	b.primitives[v] = i
	b.static.Primitives = append(b.static.Primitives, v)
	return i
}

func (b *StaticBuilder) AddDecimal(v dec128.Dec128) int {
	s := v.String()
	if i, ok := b.decimals[s]; ok {
		return i
	}
	i := len(b.static.Decimals)
	b.decimals[s] = i
	b.static.Decimals = append(b.static.Decimals, v)
	return i
}

func (b *StaticBuilder) AddString(v string) int {
	if i, ok := b.strings[v]; ok {
		return i
	}
	i := len(b.static.Strings)
	b.strings[v] = i
	b.static.Strings = append(b.static.Strings, v)
	return i
}

func (b *StaticBuilder) AddRunes(v core.Runes) int {
	s := string(v.Elements)
	if i, ok := b.runes[s]; ok {
		return i
	}
	i := len(b.static.Runes)
	b.runes[s] = i
	b.static.Runes = append(b.static.Runes, v)
	return i
}

func (b *StaticBuilder) AddFormatSpec(v core.FormatSpec) int {
	s := v.Text
	if i, ok := b.formatSpecs[s]; ok {
		return i
	}
	i := len(b.static.FormatSpecs)
	b.formatSpecs[s] = i
	b.static.FormatSpecs = append(b.static.FormatSpecs, v)
	return i
}

func (b *StaticBuilder) AddCompiledFunction(v core.CompiledFunction) int {
	s := fmt.Sprintf("%v %v %d %d %d %v %d", v.Instructions, v.SourceMap, v.NumLocals, v.MaxStack, v.NumParameters, v.VarArgs, v.NamedResult)
	if i, ok := b.compiledFunctions[s]; ok {
		return i
	}
	i := len(b.static.CompiledFunctions)
	b.compiledFunctions[s] = i
	b.static.CompiledFunctions = append(b.static.CompiledFunctions, v)
	return i
}
