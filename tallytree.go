// Package tallytree lays out a counted tree in box-drawing glyphs, with the
// counts right-aligned INSIDE the branch rules.
//
// The usual tree renderer draws its connectors and then whatever the caller
// gave it, so a column of numbers comes out ragged or needs a separate gutter
// to line up. This one stretches the rule instead: the padding between a
// branch and its count is made of the horizontal glyph, so every count lands in
// the same column and the tree still reads as one continuous line.
//
//	├──1247 All Products
//	├─┬─111 resistor
//	│ ├──67 quarter watt 5%
//	│ └───1 ceramic
//	└────34 inductor
//
// Nothing above is padded with spaces. "resistor" has three digits where the
// total has four, so its rule is one glyph longer; "ceramic" has one digit, so
// its rule is three longer. The numbers align because the rules absorb the
// difference.
//
// A FOREST, not a tree with one root. The first row above is a peer of the
// categories rather than their parent, which is what a menu usually wants: an
// "everything" entry sitting beside the divisions rather than above them.
//
// Rows is the primary call and Lines is the convenience on top of it. Rows
// hands back the prefix per node instead of finished text, because the caller
// that needed this first renders into nested HTML <details> elements with the
// glyphs as text nodes between them — it cannot use a finished string, and
// neither can anything drawing into a terminal cell grid. Lines exists because
// most callers do just want the string.
package tallytree

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// Node is one entry: a label, a count, and any children.
type Node struct {
	Label    string
	Count    int
	Children []Node
}

// GlyphSet is the box-drawing vocabulary. The zero value is unusable; leave
// Options.Glyphs zero to get DefaultGlyphs, or set individual fields to
// override them.
type GlyphSet struct {
	Horizontal string // ─ , and the padding the counts are aligned with
	Vertical   string // │
	TeeRight   string // ├
	TeeDown    string // ┬ , marking a node that has children
	CornerUR   string // └
	Blank      string // the space under a finished branch, normally " "
}

// DefaultGlyphs is the light box-drawing set.
func DefaultGlyphs() GlyphSet {
	return GlyphSet{
		Horizontal: "─",
		Vertical:   "│",
		TeeRight:   "├",
		TeeDown:    "┬",
		CornerUR:   "└",
		Blank:      " ",
	}
}

// Options controls the layout. The zero value is valid.
type Options struct {
	// Glyphs is the box-drawing set; empty fields fall back to DefaultGlyphs.
	Glyphs GlyphSet
	// Gap separates the count from the label. Defaults to a single space.
	Gap string
	// CountWidth is the column the counts are right-aligned to. Zero measures
	// the widest count in the forest, which is what makes the rules come out
	// flush on their own.
	CountWidth int
	// Depth is how many levels the structure reserves room for. Zero measures
	// the forest as it STANDS, not as it is drawn — a closed branch still costs
	// its columns.
	//
	// That is deliberate. Measuring what is drawn would let the tree shift
	// sideways every time a branch opened or closed, and would leave a closed
	// branch no column to put its TeeDown in, so a menu would lose the only
	// mark saying there is anything underneath. Set it when several forests must
	// line up with each other.
	Depth int
	// Open, when non-nil, reports whether a node's children are drawn. A node
	// whose children are hidden still shows the TeeDown that says it has some,
	// which is what makes a collapsed menu look collapsed rather than empty.
	Open func(n Node, depth int) bool
}

// Row is one node, laid out.
//
// Prefix runs from the start of the line to just before the count, rules and
// padding included, so Prefix + the count + Gap + Label is the finished line —
// and a caller wrapping the label in markup can put its own element where the
// label goes without disturbing the geometry.
type Row struct {
	Node        Node
	Depth       int
	Last        bool // last among its siblings
	HasChildren bool
	Prefix      string
	Count       int
	Label       string
}

// Line is the finished text of a row.
func (r Row) Line(opts Options) string {
	return r.Prefix + strconv.Itoa(r.Count) + gap(opts) + r.Label
}

func gap(o Options) string {
	if o.Gap == "" {
		return " "
	}
	return o.Gap
}

func (o Options) glyphs() GlyphSet {
	g, d := o.Glyphs, DefaultGlyphs()
	if g.Horizontal == "" {
		g.Horizontal = d.Horizontal
	}
	if g.Vertical == "" {
		g.Vertical = d.Vertical
	}
	if g.TeeRight == "" {
		g.TeeRight = d.TeeRight
	}
	if g.TeeDown == "" {
		g.TeeDown = d.TeeDown
	}
	if g.CornerUR == "" {
		g.CornerUR = d.CornerUR
	}
	if g.Blank == "" {
		g.Blank = d.Blank
	}
	return g
}

// Rows lays the forest out, depth first, in the order the rows are drawn.
func Rows(forest []Node, opts Options) []Row {
	g := opts.glyphs()
	width := opts.CountWidth
	if width <= 0 {
		width = countWidth(forest)
	}
	depth := opts.Depth
	if depth <= 0 {
		depth = levels(forest, 0)
	}
	var out []Row
	var walk func(nodes []Node, d int, ancestorsOpen []bool)
	walk = func(nodes []Node, d int, trailing []bool) {
		for i, n := range nodes {
			last := i == len(nodes)-1
			kids := len(n.Children) > 0
			out = append(out, Row{
				Node: n, Depth: d, Last: last, HasChildren: kids,
				Prefix: prefix(g, trailing, last, kids, d, depth, width, n.Count),
				Count:  n.Count, Label: n.Label,
			})
			if !kids || (opts.Open != nil && !opts.Open(n, d)) {
				continue
			}
			walk(n.Children, d+1, append(append([]bool(nil), trailing...), last))
		}
	}
	walk(forest, 0, nil)
	return out
}

// Lines is Rows rendered to text.
func Lines(forest []Node, opts Options) []string {
	rows := Rows(forest, opts)
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.Line(opts)
	}
	return out
}

// prefix builds the connectors and the rule padding for one row.
//
// The width before the count is the same on every row, which is what puts the
// counts in a column: each level costs two columns, the node's own branch takes
// one, and whatever is left over between the branch and the count is filled
// with the horizontal glyph. A node with children spends the first of those
// leftover columns on a TeeDown.
func prefix(g GlyphSet, trailing []bool, last, kids bool, d, depth, width, count int) string {
	var b strings.Builder
	for _, ancestorLast := range trailing {
		if ancestorLast {
			b.WriteString(g.Blank + g.Blank)
			continue
		}
		b.WriteString(g.Vertical + g.Blank)
	}
	if last {
		b.WriteString(g.CornerUR)
	} else {
		b.WriteString(g.TeeRight)
	}
	// The columns this level does not need for its own descendants, filled so
	// the count lands where every other count lands.
	spare := 2 * (depth - 1 - d)
	for i := 0; i < spare; i++ {
		if i == 1 && kids {
			b.WriteString(g.TeeDown)
			continue
		}
		b.WriteString(g.Horizontal)
	}
	b.WriteString(strings.Repeat(g.Horizontal, max(0, width-digits(count))))
	return b.String()
}

func digits(n int) int { return utf8.RuneCountInString(strconv.Itoa(n)) }

// countWidth is the widest count anywhere in the forest.
func countWidth(nodes []Node) int {
	w := 1
	var walk func([]Node)
	walk = func(ns []Node) {
		for _, n := range ns {
			w = max(w, digits(n.Count))
			walk(n.Children)
		}
	}
	walk(nodes)
	return w
}

// levels counts how deep the forest IS, open or not. See Options.Depth for why
// this ignores what is currently drawn.
func levels(nodes []Node, d int) int {
	deepest := 0
	for _, n := range nodes {
		deepest = max(deepest, d+1)
		if len(n.Children) == 0 {
			continue
		}
		deepest = max(deepest, levels(n.Children, d+1))
	}
	return deepest
}
