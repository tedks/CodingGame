package connection

import (
	"testing"
)

// FuzzGraphOperations performs random graph operations to find crashes.
func FuzzGraphOperations(f *testing.F) {
	// Seed corpus with interesting cases
	f.Add([]byte("add:a:b:0\nadd:b:c:1\nadd:c:a:2\ndetect\n"))                     // Cycle
	f.Add([]byte("add:a:a:0\ndetect\n"))                                           // Self-loop
	f.Add([]byte("add:a:b:0\nremove:a:b:0\n"))                                     // Add then remove
	f.Add([]byte("add:a:b:0\nadd:a:b:0\n"))                                        // Duplicate
	f.Add([]byte("clear\nadd:x:y:0\n"))                                            // Clear then add
	f.Add([]byte("add::b:0\n"))                                                    // Empty from
	f.Add([]byte("add:a::0\n"))                                                    // Empty to
	f.Add([]byte("add:a:b:0\nadd:b:c:0\nadd:c:d:0\nremove:b:c:0\ndetect\n"))       // Remove middle
	f.Add([]byte("add:ファイル:出力:0\n"))                                          // Unicode
	f.Add([]byte("add:a:b:0\nadd:b:a:1\nadd:c:d:0\nadd:d:c:1\ndetect\n"))          // Two cycles

	f.Fuzz(func(t *testing.T, data []byte) {
		g := NewGraph()
		operations := parseOperations(data)

		for _, op := range operations {
			switch op.cmd {
			case "add":
				g.AddNew(op.from, op.to, op.typ)
			case "remove":
				id := makeID(op.from, op.to, op.typ)
				g.Remove(id)
			case "detect":
				g.DetectCircular()
			case "clear":
				g.Clear()
			case "count":
				_ = g.Count()
			case "all":
				_ = g.All()
			case "fanin":
				_ = g.FanIn(op.from)
			case "fanout":
				_ = g.FanOut(op.to)
			case "fromfile":
				_ = g.FromFile(op.from)
			case "tofile":
				_ = g.ToFile(op.to)
			}
		}

		// Verify invariants hold at the end
		count := g.Count()
		all := g.All()
		if count != len(all) {
			t.Errorf("Count()=%d != len(All())=%d", count, len(all))
		}
	})
}

// operation represents a parsed fuzzer operation.
type operation struct {
	cmd  string
	from string
	to   string
	typ  Type
}

// parseOperations parses fuzzer input into operations.
// Format: "cmd:from:to:type\n" where type is 0-4
func parseOperations(data []byte) []operation {
	var ops []operation
	var current []byte
	var parts []string

	for _, b := range data {
		if b == '\n' {
			if len(current) > 0 {
				parts = splitBytes(current, ':')
				if len(parts) >= 1 {
					op := operation{cmd: parts[0]}
					if len(parts) >= 2 {
						op.from = parts[1]
					}
					if len(parts) >= 3 {
						op.to = parts[2]
					}
					if len(parts) >= 4 {
						op.typ = parseType(parts[3])
					}
					ops = append(ops, op)
				}
			}
			current = nil
		} else {
			current = append(current, b)
		}
	}
	// Handle last line without newline
	if len(current) > 0 {
		parts = splitBytes(current, ':')
		if len(parts) >= 1 {
			op := operation{cmd: parts[0]}
			if len(parts) >= 2 {
				op.from = parts[1]
			}
			if len(parts) >= 3 {
				op.to = parts[2]
			}
			if len(parts) >= 4 {
				op.typ = parseType(parts[3])
			}
			ops = append(ops, op)
		}
	}

	return ops
}

// splitBytes splits bytes by a delimiter, returning strings.
func splitBytes(data []byte, delim byte) []string {
	var parts []string
	var current []byte

	for _, b := range data {
		if b == delim {
			parts = append(parts, string(current))
			current = nil
		} else {
			current = append(current, b)
		}
	}
	parts = append(parts, string(current))
	return parts
}

// parseType parses a type string to Type.
func parseType(s string) Type {
	if len(s) == 0 {
		return TypeImport
	}
	switch s[0] {
	case '0':
		return TypeImport
	case '1':
		return TypeInheritance
	case '2':
		return TypeComposition
	case '3':
		return TypeCall
	default:
		return TypeUnknown
	}
}

// FuzzConnectionPaths fuzzes connection creation with random paths.
func FuzzConnectionPaths(f *testing.F) {
	// Seed with interesting paths
	f.Add("a.go", "b.go")
	f.Add("", "b.go")
	f.Add("a.go", "")
	f.Add("", "")
	f.Add("a:b:c", "d:e:f")              // Colons in paths
	f.Add("a\x1Fb", "c\x1Fd")            // Unit separator in paths
	f.Add("ファイル.go", "出力.go")       // Japanese
	f.Add("file with spaces.go", "out") // Spaces
	f.Add("file\ttab.go", "out")        // Tab
	f.Add("file\nnewline.go", "out")    // Newline
	f.Add("file\x00null.go", "out")     // Null byte
	f.Add(string(make([]byte, 1000)), "b.go") // Long path

	f.Fuzz(func(t *testing.T, from, to string) {
		// Create connection - should not panic
		c := NewConnection(from, to, TypeImport)

		// Verify basic properties
		if c == nil {
			t.Error("NewConnection returned nil")
			return
		}

		// ID should be stable
		id1 := c.ID()
		id2 := c.ID()
		if id1 != id2 {
			t.Error("ID() is not stable")
		}

		// Endpoints should match input (after normalization)
		// Note: backslashes are normalized to forward slashes
		gotFrom := c.From()
		gotTo := c.To()
		if len(gotFrom) != len(normalizeForComparison(from)) {
			t.Errorf("From length mismatch: got %d, want %d", len(gotFrom), len(from))
		}
		if len(gotTo) != len(normalizeForComparison(to)) {
			t.Errorf("To length mismatch: got %d, want %d", len(gotTo), len(to))
		}

		// Graph operations should work
		g := NewGraph()
		added := g.Add(c)
		if added == nil {
			t.Error("Graph.Add returned nil")
		}
		if g.Count() != 1 {
			t.Errorf("Count after add = %d, want 1", g.Count())
		}

		// Get by ID should work
		retrieved := g.Get(id1)
		if retrieved != c {
			t.Error("Get by ID failed")
		}

		// GetByEndpoints should work
		retrieved2 := g.GetByEndpoints(gotFrom, gotTo, TypeImport)
		if retrieved2 != c {
			t.Error("GetByEndpoints failed")
		}

		// DetectCircular should not panic
		g.DetectCircular()

		// Remove should work
		if !g.Remove(id1) {
			t.Error("Remove failed")
		}
		if g.Count() != 0 {
			t.Errorf("Count after remove = %d, want 0", g.Count())
		}
	})
}

// normalizeForComparison normalizes a path for comparison (same as in connection.go).
func normalizeForComparison(p string) string {
	result := make([]byte, len(p))
	for i := 0; i < len(p); i++ {
		if p[i] == '\\' {
			result[i] = '/'
		} else {
			result[i] = p[i]
		}
	}
	return string(result)
}

// FuzzMakeID fuzzes ID generation to ensure no collisions.
func FuzzMakeID(f *testing.F) {
	f.Add("a", "b", uint8(0))
	f.Add("a:b", "c", uint8(0))
	f.Add("a", "b:c", uint8(0))
	f.Add("", "", uint8(0))
	f.Add("\x1F", "\x1F", uint8(0))

	seen := make(map[string]struct {
		from string
		to   string
		typ  Type
	})

	f.Fuzz(func(t *testing.T, from, to string, typByte uint8) {
		typ := Type(typByte % 5)
		id := makeID(from, to, typ)

		// Check for collision
		if prev, exists := seen[id]; exists {
			if prev.from != from || prev.to != to || prev.typ != typ {
				t.Errorf("ID collision!\n"+
					"ID: %q\n"+
					"prev: from=%q, to=%q, typ=%v\n"+
					"curr: from=%q, to=%q, typ=%v",
					id, prev.from, prev.to, prev.typ, from, to, typ)
			}
		} else {
			seen[id] = struct {
				from string
				to   string
				typ  Type
			}{from, to, typ}
		}
	})
}
