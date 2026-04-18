// Package templates will own template CRUD + publish in W2.
// W1 scaffolds the package so import graph compiles.
package templates

// Module is a placeholder; real wiring lands in W2 plan.
type Module struct{}

func New() *Module { return &Module{} }
