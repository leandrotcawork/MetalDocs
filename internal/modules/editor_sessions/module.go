// Package editor_sessions will own pessimistic editor locks in W3.
package editor_sessions

type Module struct{}

func New() *Module { return &Module{} }
