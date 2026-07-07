package compiler

import "github.com/jokruger/kavun/parser"

type OptimizationConfig struct {
	MaxPasses int
}

func (oc *OptimizationConfig) SetO0() {
	oc.MaxPasses = 0
}

func (oc *OptimizationConfig) SetO1() {
	oc.MaxPasses = 1
}

func (oc *OptimizationConfig) SetO2() {
	oc.MaxPasses = 1
}

func (oc *OptimizationConfig) SetO3() {
	oc.MaxPasses = 10
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

func (c *Compiler) Optimize(node parser.Node) (parser.Node, error) {
	if c.oc == nil || c.oc.MaxPasses <= 0 {
		return node, nil
	}

	var err error
	var changed bool
	pass := 0
	for {
		node, changed, err = c.optimize(node)
		if err != nil {
			return nil, err
		}
		if !changed {
			break
		}
		pass++
		if pass >= c.oc.MaxPasses {
			break
		}
	}

	return node, nil
}

func (c *Compiler) optimize(node parser.Node) (parser.Node, bool, error) {
	return node, false, nil
}
