package domain

import "testing"

func TestValuesHash_OrderIndependent(t *testing.T) {
	a := map[string]any{"p1": "x", "p2": "y"}
	b := map[string]any{"p2": "y", "p1": "x"}
	if ComputeValuesHash(a) != ComputeValuesHash(b) {
		t.Fatal("hash must be order-independent")
	}
}

func TestValuesHash_ChangesOnValueChange(t *testing.T) {
	if ComputeValuesHash(map[string]any{"p1": "x"}) == ComputeValuesHash(map[string]any{"p1": "y"}) {
		t.Fatal("hash must differ on value change")
	}
}
