package basicnode

import (
	"fmt"
	"reflect"

	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/node/mixins"
)

var (
	_ datamodel.Node                         = &plainLink{}
	_ datamodel.NodePrototype                = Prototype__Link{}
	_ datamodel.NodePrototypeSupportingAmend = Prototype__Link{}
	_ datamodel.NodeBuilder                  = &plainLink__Builder{}
	_ datamodel.NodeAssembler                = &plainLink__Assembler{}
)

func NewLink(value datamodel.Link) datamodel.Node {
	return &plainLink{x: value}
}

// plainLink is a simple box around a Link that complies with datamodel.Node.
type plainLink struct {
	x datamodel.Link
	c datamodel.NodeAmender
}

// -- Node interface methods -->

func (plainLink) Kind() datamodel.Kind {
	return datamodel.Kind_Link
}
func (plainLink) LookupByString(string) (datamodel.Node, error) {
	return mixins.Link{TypeName: "link"}.LookupByString("")
}
func (plainLink) LookupByNode(key datamodel.Node) (datamodel.Node, error) {
	return mixins.Link{TypeName: "link"}.LookupByNode(nil)
}
func (plainLink) LookupByIndex(idx int64) (datamodel.Node, error) {
	return mixins.Link{TypeName: "link"}.LookupByIndex(0)
}
func (plainLink) LookupBySegment(seg datamodel.PathSegment) (datamodel.Node, error) {
	return mixins.Link{TypeName: "link"}.LookupBySegment(seg)
}
func (plainLink) MapIterator() datamodel.MapIterator {
	return nil
}
func (plainLink) ListIterator() datamodel.ListIterator {
	return nil
}
func (plainLink) Length() int64 {
	return -1
}
func (plainLink) IsAbsent() bool {
	return false
}
func (plainLink) IsNull() bool {
	return false
}
func (plainLink) AsBool() (bool, error) {
	return mixins.Link{TypeName: "link"}.AsBool()
}
func (plainLink) AsInt() (int64, error) {
	return mixins.Link{TypeName: "link"}.AsInt()
}
func (plainLink) AsFloat() (float64, error) {
	return mixins.Link{TypeName: "link"}.AsFloat()
}
func (plainLink) AsString() (string, error) {
	return mixins.Link{TypeName: "link"}.AsString()
}
func (plainLink) AsBytes() ([]byte, error) {
	return mixins.Link{TypeName: "link"}.AsBytes()
}
func (n *plainLink) AsLink() (datamodel.Link, error) {
	return n.x, nil
}
func (plainLink) Prototype() datamodel.NodePrototype {
	return Prototype__Link{}
}

// -- NodePrototype -->

type Prototype__Link struct{}

func (Prototype__Link) NewBuilder() datamodel.NodeBuilder {
	var w plainLink
	return &plainLink__Builder{plainLink__Assembler{w: &w}}
}

// -- NodePrototypeSupportingAmend -->

func (p Prototype__Link) AmendingBuilder(base datamodel.Node) datamodel.NodeAmender {
	var l *plainLink
	if base != nil {
		// If `base` is specified, it MUST be another `plainLink`.
		if baseLink, castOk := base.(*plainLink); !castOk {
			panic("misuse")
		} else {
			l = baseLink
		}
	} else {
		l = &plainLink{}
	}
	return p.amender(l)
}

func (Prototype__Link) amender(base datamodel.Node) datamodel.NodeAmender {
	return &plainLink__Builder{plainLink__Assembler{w: base.(*plainLink)}}
}

// -- NodeBuilder -->

type plainLink__Builder struct {
	plainLink__Assembler
}

func (nb *plainLink__Builder) Build() datamodel.Node {
	return nb.w
}
func (nb *plainLink__Builder) Reset() {
	var w plainLink
	*nb = plainLink__Builder{plainLink__Assembler{w: &w}}
}

// -- NodeAmender -->

func (nb *plainLink__Builder) Get(cfg datamodel.NodeAmendCfg, path datamodel.Path) (datamodel.Node, error) {
	err := nb.loadLink(cfg)
	if err != nil {
		return nil, err
	}
	if path.Len() == 0 {
		return nb.Build(), nil
	}
	return nb.w.c.Get(cfg, path)
}

func (nb *plainLink__Builder) Transform(cfg datamodel.NodeAmendCfg, path datamodel.Path, transform func(datamodel.Node) (datamodel.Node, error), createParents bool) (datamodel.Node, error) {
	// Allow the base node to be replaced.
	if path.Len() == 0 {
		prevNode := nb.Build()
		if newNode, err := transform(prevNode); err != nil {
			return nil, err
		} else if newLink, castOk := newNode.(*plainLink); !castOk { // only use another `plainLink` to replace this one
			return nil, fmt.Errorf("transform: cannot transform root into incompatible type: %v", reflect.TypeOf(newNode))
		} else {
			*nb.w = *newLink
			return prevNode, nil
		}
	}
	err := nb.loadLink(cfg)
	if err != nil {
		return nil, err
	}
	childVal, err := nb.w.c.Transform(cfg, path, transform, createParents)
	if err != nil {
		return nil, err
	}
	newLink, err := cfg.LinkStorer(nb.w.x.Prototype(), nb.Build())
	if err != nil {
		return nil, fmt.Errorf("transform: error storing transformed node at %q: %w", path, err)
	}
	nb.w.x = newLink
	return childVal, nil
}

func (nb *plainLink__Builder) loadLink(cfg datamodel.NodeAmendCfg) error {
	if nb.w.c == nil {
		c, err := cfg.LinkLoader(nb.w.x)
		if err != nil {
			return err
		}
		nb.w.c = NewAmender(c, c.Kind())
	}
	return nil
}

// -- NodeAssembler -->

type plainLink__Assembler struct {
	w *plainLink
}

func (plainLink__Assembler) BeginMap(sizeHint int64) (datamodel.MapAssembler, error) {
	return mixins.LinkAssembler{TypeName: "link"}.BeginMap(0)
}
func (plainLink__Assembler) BeginList(sizeHint int64) (datamodel.ListAssembler, error) {
	return mixins.LinkAssembler{TypeName: "link"}.BeginList(0)
}
func (plainLink__Assembler) AssignNull() error {
	return mixins.LinkAssembler{TypeName: "link"}.AssignNull()
}
func (plainLink__Assembler) AssignBool(bool) error {
	return mixins.LinkAssembler{TypeName: "link"}.AssignBool(false)
}
func (plainLink__Assembler) AssignInt(int64) error {
	return mixins.LinkAssembler{TypeName: "link"}.AssignInt(0)
}
func (plainLink__Assembler) AssignFloat(float64) error {
	return mixins.LinkAssembler{TypeName: "link"}.AssignFloat(0)
}
func (plainLink__Assembler) AssignString(string) error {
	return mixins.LinkAssembler{TypeName: "link"}.AssignString("")
}
func (plainLink__Assembler) AssignBytes([]byte) error {
	return mixins.LinkAssembler{TypeName: "link"}.AssignBytes(nil)
}
func (na *plainLink__Assembler) AssignLink(v datamodel.Link) error {
	na.w.x = v
	return nil
}
func (na *plainLink__Assembler) AssignNode(v datamodel.Node) error {
	if v2, err := v.AsLink(); err != nil {
		return err
	} else {
		na.w.x = v2
		return nil
	}
}
func (plainLink__Assembler) Prototype() datamodel.NodePrototype {
	return Prototype__Link{}
}
