package compiler

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

func (c *OptimizationConfig) SetO0() {
}

func (c *OptimizationConfig) SetO1() {
}

func (c *OptimizationConfig) SetO2() {
}

func (c *OptimizationConfig) SetO3() {
}
