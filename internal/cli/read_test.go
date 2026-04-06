package cli

import "testing"

func TestReadCommandsBindShapeFlagsOnlyOnReadCommands(t *testing.T) {
	root := newRootCommand()

	tests := []struct {
		path        []string
		expectShape bool
	}{
		{path: []string{"cmdb", "get"}, expectShape: true},
		{path: []string{"cmdb", "list"}, expectShape: true},
		{path: []string{"monitor", "get"}, expectShape: true},
		{path: []string{"system", "interfaces"}, expectShape: true},
		{path: []string{"system", "status"}, expectShape: false},
		{path: []string{"system", "backup"}, expectShape: false},
	}

	for _, tc := range tests {
		cmd, _, err := root.Find(tc.path)
		if err != nil {
			t.Fatalf("Find(%v) returned error: %v", tc.path, err)
		}
		hasShape := cmd.Flags().Lookup("select") != nil
		if hasShape != tc.expectShape {
			t.Fatalf("%v shape flags = %v, want %v", tc.path, hasShape, tc.expectShape)
		}
	}
}
