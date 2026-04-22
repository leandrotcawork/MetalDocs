//go:build integration

package domain_test

import "testing"

func TestSequenceCounter_NoDuplicates(t *testing.T) {
	t.Skip("requires live DB - run with -tags=integration against a seeded postgres")
}
