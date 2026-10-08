# tallytree

Lays out a counted tree in box-drawing glyphs, with the counts right-aligned
**inside the branch rules**.

```
├──1247 All Products
├─┬─111 resistor
│ ├──67 quarter watt 5%
│ └───1 ceramic
└────34 inductor
```

Nothing there is padded with spaces. `resistor` has three digits where the total
has four, so its rule runs one glyph longer; `ceramic` has one digit, so its rule
runs three longer. The numbers line up because the rules absorb the difference,
and the tree still reads as one continuous line instead of a tree and a column
with a gutter between them.

It is a **forest**, not a tree with one root. The first row above is a peer of
the categories rather than their parent, which is what a menu usually wants: an
"everything" entry beside the divisions rather than above them.

## Use

```go
rows := tallytree.Rows([]tallytree.Node{
    {Label: "All Products", Count: 1247},
    {Label: "resistor", Count: 111, Children: []tallytree.Node{
        {Label: "quarter watt 5%", Count: 67},
        {Label: "ceramic", Count: 1},
    }},
    {Label: "inductor", Count: 34},
}, tallytree.Options{})

for _, r := range rows {
    fmt.Println(r.Prefix, r.Count, r.Label)
}
```

`Lines` is the same thing rendered to strings.

## Why Rows hands back a prefix

`Rows` gives you the connectors and the padding, and stops before the count. It
does not hand back finished text, because the caller it was extracted from
cannot use finished text: that menu is nested HTML `<details>` elements with the
glyphs as text nodes between them, and each label is an `<a>`. A string would
have to be taken apart again.

The same is true of anything drawing into a terminal cell grid, where the label
is styled differently from the rule. So `Prefix + count + Gap + Label` is the
line if you want it, and `Prefix` is the part you keep if you are going to write
the rest yourself.

## Options

- **Gap** separates the count from the label. A single space by default; the
  HTML caller passes a non-breaking space.
- **CountWidth** is the column the counts align to. Zero measures the widest
  count, which is what makes the rules come out flush on their own.
- **Depth** is how many levels the structure reserves room for. Zero measures the
  forest as it stands — *not* as it is drawn, so a closed branch still costs its
  columns and keeps the `┬` that says there is something under it. Pin it when a
  tree that is genuinely flat should still leave room, or when several forests
  must line up with each other.
- **Open** decides whether a node's children are drawn, for a menu that expands.
- **Glyphs** replaces the box-drawing set.

## Where it came from

The category menu on [magnetosphere.net](https://magnetosphere.net), which drew
it twice: once in an `html/template` with `repeat` and `sub`, and once again in
Go for the terminal UI, whose comment read "reproduces htmpl/catsubcats.html".
Two implementations of one drawing, kept in agreement by hand and tested by
neither.

When this code was extracted, its output was compared with both originals: it
matched the Go terminal one byte for byte, and the glyph runs it predicts
appeared verbatim in the served HTML of the other. Those comparisons were made
against the original implementations; this repo's own tests (`tallytree_test.go`)
do not repeat them.
