package core

// A widget: the fixture for a type declared in package core.
//
// ## Overview
//
// The page head: its own `##` sections, verbatim.
//
// - a list item
// with an unindented continuation line
// - a second item
//
// | operator | reading |
// | --- | --- |
// | `w + x` | exactly `append(x)` |
//
// ### A third-level heading
//
// ```go
// w := widget()
//
// w.len()   // 0
// ```
//
//kavun:type widget

// TypeWidget is the widget's descriptor; its table is generated.
var TypeWidget = ValueTypeDescr{
	Methods: widgetMethods,
}
