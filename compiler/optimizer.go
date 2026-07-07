package compiler

import "github.com/jokruger/kavun/parser"

type OptimizationConfig struct {
}

func O0() *OptimizationConfig {
	oc := &OptimizationConfig{}
	oc.SetO0()
	return oc
}

func O1() *OptimizationConfig {
	oc := &OptimizationConfig{}
	oc.SetO1()
	return oc
}

func O2() *OptimizationConfig {
	oc := &OptimizationConfig{}
	oc.SetO2()
	return oc
}

func O3() *OptimizationConfig {
	oc := &OptimizationConfig{}
	oc.SetO3()
	return oc
}

func (oc *OptimizationConfig) SetO0() {
}

func (oc *OptimizationConfig) SetO1() {
}

func (oc *OptimizationConfig) SetO2() {
}

func (oc *OptimizationConfig) SetO3() {
}

func (c *Compiler) Optimize(node parser.Node) (parser.Node, error) {
	return node, nil
}
