package repository

import (
	"os"
	"strings"
	"testing"
)

func TestCreateDocumentInsertIncludesBridgeSnapshots(t *testing.T) {
	src, err := os.ReadFile("repository.go")
	if err != nil {
		t.Fatalf("read repository.go: %v", err)
	}

	body := string(src)
	start := strings.Index(body, "func (r *Repository) CreateDocument")
	if start == -1 {
		t.Fatal("CreateDocument not found")
	}
	end := strings.Index(body[start:], "func (r *Repository) SetRevisionStorageKey")
	if end == -1 {
		t.Fatal("SetRevisionStorageKey not found")
	}
	createDocument := body[start : start+end]

	tests := []struct {
		name string
		want string
	}{
		{name: "profile column", want: "profile_code_snapshot"},
		{name: "process area column", want: "process_area_code_snapshot"},
		{name: "profile arg", want: "d.ProfileCodeSnapshot"},
		{name: "process area arg", want: "d.ProcessAreaCodeSnapshot"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(createDocument, tt.want) {
				t.Fatalf("CreateDocument missing %s", tt.want)
			}
		})
	}
}
