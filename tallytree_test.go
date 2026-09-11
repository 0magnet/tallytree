package tallytree

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// shop is the catalogue shape this was extracted from: an "everything" entry
// beside the categories, one of which is open.
func shop() []Node {
	return []Node{
		{Label: "All Products", Count: 1247},
		{Label: "resistor", Count: 111, Children: []Node{
			{Label: "quarter watt 5%", Count: 67},
			{Label: "ceramic", Count: 1},
		}},
		{Label: "inductor", Count: 34},
	}
}

// The layout has to match magnetosphere.net's category menu exactly — both the
// template that draws it in HTML and the Go that draws it in the terminal.
// These five lines were taken from the running code before any of this existed.
func TestMatchesTheShopMenu(t *testing.T) {
	want := []string{
		"├──1247 All Products",
		"├─┬─111 resistor",
		"│ ├──67 quarter watt 5%",
		"│ └───1 ceramic",
		"└────34 inductor",
	}
	got := Lines(shop(), Options{})
	if len(got) != len(want) {
		t.Fatalf("%d rows, want %d:\n%s", len(got), len(want), strings.Join(got, "\n"))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d:\n got %q\nwant %q", i, got[i], want[i])
		}
	}
}

// The whole point: every count ends in the same column, whatever its width.
func TestCountsLandInOneColumn(t *testing.T) {
	for _, forest := range [][]Node{
		shop(),
		{{Label: "a", Count: 1}, {Label: "b", Count: 1000000}},
		{{Label: "deep", Count: 5, Children: []Node{
			{Label: "deeper", Count: 40, Children: []Node{{Label: "deepest", Count: 300}}},
		}}},
	} {
		var w int
		for i, r := range Rows(forest, Options{}) {
			n := utf8.RuneCountInString(r.Prefix) + utf8.RuneCountInString(itoa(r.Count))
			if i == 0 {
				w = n
				continue
			}
			if n != w {
				t.Errorf("row %d ends its count at column %d, the first ended at %d", i, n, w)
			}
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

// A closed branch keeps its ┬ — that is what says there is something under it —
// but costs no columns, so the tree does not shift as branches open and close
// unless the caller pins Depth.
func TestClosedBranchKeepsItsTee(t *testing.T) {
	closed := Lines(shop(), Options{Open: func(Node, int) bool { return false }})
	if len(closed) != 3 {
		t.Fatalf("a closed menu should be 3 rows, got %d:\n%s", len(closed), strings.Join(closed, "\n"))
	}
	if !strings.Contains(closed[1], "┬") {
		t.Errorf("the closed category lost the mark saying it has children: %q", closed[1])
	}
	// Pinned depth keeps the columns where the open tree put them.
	pinned := Lines(shop(), Options{Depth: 2, Open: func(Node, int) bool { return false }})
	open := Lines(shop(), Options{})
	if a, b := colOf(pinned[0]), colOf(open[0]); a != b {
		t.Errorf("pinned depth moved the count column: %d vs %d", a, b)
	}
}

func colOf(line string) int { return strings.Index(line, " ") }

// Rows exists so a caller can put its own markup where the label goes; the
// prefix must be usable on its own.
func TestPrefixIsTheLineWithoutTheLabel(t *testing.T) {
	for _, r := range Rows(shop(), Options{}) {
		if line := r.Line(Options{}); !strings.HasPrefix(line, r.Prefix) {
			t.Errorf("prefix %q is not the start of %q", r.Prefix, line)
		}
	}
}
