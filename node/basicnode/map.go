package basicnode

import (
	"fmt"
	"reflect"

	"github.com/emirpasic/gods/maps/linkedhashmap"

	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/node/mixins"
)

var (
	_ datamodel.Node                         = &plainMap{}
	_ datamodel.NodePrototype                = Prototype__Map{}
	_ datamodel.NodePrototypeSupportingAmend = Prototype__Map{}
	_ datamodel.NodeAmender                  = &plainMap__Builder{}
	_ datamodel.NodeAssembler                = &plainMap__Assembler{}
)

// plainMap is a concrete type that provides a map-kind datamodel.Node.
// It can contain any kind of value.
// plainMap is also embedded in the 'any' struct and usable from there.
type plainMap struct {
	// Parent node (can be any recursive type)
	p datamodel.NodeAmender
	// Map contents
	m linkedhashmap.Map
	// The following fields are needed to present an accurate "effective" view of the base node and all accumulated
	// updates. These fields don't need to be used unless a base node was specified and we have an existing node to
	// update.
	//
	// Base node (must be a `plainMap`)
	b *plainMap
	// This is the count of children *present in the base node* that are removed. Knowing this count allows accurate
	// traversal of the "effective" node view.
	r int
	// This is the count of new children. If an added node is removed, this count should be decremented instead of `r`.
	a int
}

type plainMap__Entry struct {
	k plainString           // address of this used when we return keys as nodes, such as in iterators.  Need in one place to amortize shifts to heap when ptr'ing for iface.
	v datamodel.NodeAmender // store an amender instead of the node so that we can access both the node and the API to update it.
	a bool                  // whether this node was "added" to the map, or wraps an existing base node child node.
}

// -- Node interface methods -->

func (plainMap) Kind() datamodel.Kind {
	return datamodel.Kind_Map
}
func (n *plainMap) LookupByString(key string) (datamodel.Node, error) {
	if entry, err := n.lookupEntryByString(key); err != nil {
		return nil, err
	} else {
		return entry.v.Build(), err
	}
}
func (n *plainMap) lookupEntryByString(key string) (*plainMap__Entry, error) {
	// Look at local state first
	if entry, exists := n.m.Get(key); exists {
		e := entry.(*plainMap__Entry)
		if e.v == nil {
			// Node was removed
			return nil, datamodel.ErrNotExists{Segment: datamodel.PathSegmentOfString(key)}
		}
		return e, nil
	}
	// Fallback to base state (if available)
	if n.b != nil {
		if entry, exists := n.b.m.Get(key); exists {
			e := entry.(*plainMap__Entry)
			return e, nil
		}
	}
	return nil, datamodel.ErrNotExists{Segment: datamodel.PathSegmentOfString(key)}
}
func (n *plainMap) LookupByNode(key datamodel.Node) (datamodel.Node, error) {
	ks, err := key.AsString()
	if err != nil {
		return nil, err
	}
	return n.LookupByString(ks)
}
func (plainMap) LookupByIndex(idx int64) (datamodel.Node, error) {
	return mixins.Map{TypeName: "map"}.LookupByIndex(0)
}
func (n *plainMap) LookupBySegment(seg datamodel.PathSegment) (datamodel.Node, error) {
	return n.LookupByString(seg.String())
}
func (n *plainMap) MapIterator() datamodel.MapIterator {
	var b datamodel.MapIterator = nil
	// If all children were removed from the base node, or no base node was specified, there is nothing to iterate
	// over w.r.t. that node.
	if (n.b != nil) && (int64(n.r) < n.b.Length()) {
		b = n.b.MapIterator()
	}
	var m *linkedhashmap.Iterator
	if (n.r != 0) || (n.a != 0) {
		itr := n.m.Iterator()
		m = &itr
	}
	return &plainMap_MapIterator{n, m, b, 0}
}
func (plainMap) ListIterator() datamodel.ListIterator {
	return nil
}
func (n *plainMap) Length() int64 {
	length := int64(n.a - n.r)
	if n.b != nil {
		length = length + n.b.Length()
	}
	return length
}
func (plainMap) IsAbsent() bool {
	return false
}
func (plainMap) IsNull() bool {
	return false
}
func (plainMap) AsBool() (bool, error) {
	return mixins.Map{TypeName: "map"}.AsBool()
}
func (plainMap) AsInt() (int64, error) {
	return mixins.Map{TypeName: "map"}.AsInt()
}
func (plainMap) AsFloat() (float64, error) {
	return mixins.Map{TypeName: "map"}.AsFloat()
}
func (plainMap) AsString() (string, error) {
	return mixins.Map{TypeName: "map"}.AsString()
}
func (plainMap) AsBytes() ([]byte, error) {
	return mixins.Map{TypeName: "map"}.AsBytes()
}
func (plainMap) AsLink() (datamodel.Link, error) {
	return mixins.Map{TypeName: "map"}.AsLink()
}
func (plainMap) Prototype() datamodel.NodePrototype {
	return Prototype.Map
}

type plainMap_MapIterator struct {
	n   *plainMap
	m   *linkedhashmap.Iterator
	b   datamodel.MapIterator
	idx int
}

func (itr *plainMap_MapIterator) Next() (k datamodel.Node, v datamodel.Node, _ error) {
	if itr.Done() {
		return nil, nil, datamodel.ErrIteratorOverread{}
	}
	if itr.b != nil {
		// Iterate over base node first to maintain ordering.
		var err error
		for !itr.b.Done() {
			k, v, err = itr.b.Next()
			if err != nil {
				return nil, nil, err
			}
			ks, _ := k.AsString()
			if err != nil {
				return nil, nil, err
			}
			if entry, exists := itr.n.m.Get(ks); exists {
				e := entry.(*plainMap__Entry)
				// Skip removed nodes
				if e.v == nil {
					continue
				}
				v = e.v.Build()
				// Fall-through and return wrapped nodes
			}
			// We found a "real" node to return, increment the counter.
			itr.idx++
			return
		}
	}
	if itr.m != nil {
		// Iterate over mods, skipping removed nodes.
		for itr.m.Next() {
			entry := itr.m.Value().(*plainMap__Entry)
			// Skip removed nodes
			if entry.v == nil {
				continue
			}
			k = &entry.k
			v = entry.v.Build()
			// Skip "wrapper" nodes that represent existing sub-nodes in the hierarchy corresponding to an added leaf
			// node.
			if !entry.a {
				continue
			}
			// We found a "real" node to return, increment the counter.
			itr.idx++
			return
		}
	}
	return nil, nil, datamodel.ErrIteratorOverread{}
}
func (itr *plainMap_MapIterator) Done() bool {
	return int64(itr.idx) >= itr.n.Length()
}

// -- NodePrototype -->

type Prototype__Map struct{}

func (p Prototype__Map) NewBuilder() datamodel.NodeBuilder {
	return p.AmendingBuilder(nil)
}

// -- NodePrototypeSupportingAmend -->

func (Prototype__Map) AmendingBuilder(base datamodel.Node) datamodel.NodeAmender {
	var b *plainMap = nil
	if base != nil {
		// If `base` is specified, it MUST be another `plainMap`.
		if baseMap, castOk := base.(*plainMap); !castOk {
			panic("misuse")
		} else {
			b = baseMap
		}
	}
	return &plainMap__Builder{plainMap__Assembler{w: &plainMap{b: b}}}
}

// -- NodeBuilder -->

type plainMap__Builder struct {
	plainMap__Assembler
}

func (nb *plainMap__Builder) Build() datamodel.Node {
	if (nb.state != maState_initial) && (nb.state != maState_finished) {
		panic("invalid state: assembly in progress must be 'finished' before Build can be called!")
	}
	return nb.w
}
func (nb *plainMap__Builder) Reset() {
	base := nb.w.b
	*nb = plainMap__Builder{}
	nb.w = &plainMap{b: base}
}

// -- NodeAmender -->

func (nb *plainMap__Builder) Get(path datamodel.Path) (datamodel.Node, error) {
	// If the root is requested, return the `Node` view of the amender.
	if path.Len() == 0 {
		return nb.w, nil
	}
	childSeg, remainingPath := path.Shift()
	childKey := childSeg.String()
	childEntry, err := nb.w.lookupEntryByString(childKey)
	// Since we're explicitly looking for a node, look for the child node in the current amender state and throw an
	// error if it does not exist.
	if err != nil {
		return nil, err
	}
	childVal := childEntry.v.Build()
	return nb.storeChildEntry(childKey, childVal, childVal.Kind(), false).Get(remainingPath)
}

func (nb *plainMap__Builder) Transform(path datamodel.Path, transform func(datamodel.Node) (datamodel.Node, error), createParents bool) (datamodel.Node, error) {
	// Allow the base node to be replaced.
	if path.Len() == 0 {
		prevNode := nb.w
		if newNode, err := transform(prevNode); err != nil {
			return nil, err
		} else if newMap, castOk := newNode.(*plainMap); !castOk { // only use another `plainMap` to replace this one
			return nil, fmt.Errorf("map transform: cannot transform root into incompatible type: %v", reflect.TypeOf(newNode))
		} else {
			*nb.w = *newMap
			return prevNode, nil
		}
	}
	childSeg, remainingPath := path.Shift()
	childKey := childSeg.String()
	atLeaf := remainingPath.Len() == 0
	childEntry, err := nb.w.lookupEntryByString(childKey)
	if err != nil {
		// - Return any error other than "not exists".
		// - If the child node does not exist and `createParents = true`, create the new hierarchy, otherwise throw an
		//   error.
		// - Even if `createParent = false`, if we're at the leaf, don't throw an error because we don't need to create
		//   any more intermediate parent nodes.
		if _, notFoundErr := err.(datamodel.ErrNotExists); !notFoundErr || !(atLeaf || createParents) {
			return nil, fmt.Errorf("transform: parent position at %q did not exist (and createParents was false)", path)
		}
	}
	if atLeaf {
		if newChildVal, err := transform(childEntry.v.Build()); err != nil {
			return nil, err
		} else if newChildVal == nil {
			// Use the "Null" node to indicate a removed child.
			nb.w.m.Put(childKey, &plainMap__Entry{plainString(childKey), nil, false})
			// If the child being removed didn't already exist, we could error out but we don't have to because the
			// state will remain consistent. This operation is equivalent to adding a child then removing it, in which
			// case we would have incremented then decremented `adds`, leaving it the same.
			if childEntry != nil {
				// If the child node being removed is a new node previously added to the node hierarchy, decrement
				// `adds`, otherwise increment `rems`. This allows us to retain knowledge about the "history" of the
				// base hierarchy.
				if childEntry.a {
					nb.w.a--
				} else {
					nb.w.r++
				}
			}
		} else {
			// While building the nested amender tree, only count nodes as "added" when they didn't exist and had to be
			// created to fill out the hierarchy.
			created := false
			if childEntry == nil {
				nb.w.a++
				created = true
			}
			nb.storeChildEntry(childKey, newChildVal, newChildVal.Kind(), created)
		}
		return childEntry.v.Build(), nil
	}
	// While building the nested amender tree, only count nodes as "added" when they didn't exist and had to be created
	// to fill out the hierarchy.
	var childKind datamodel.Kind
	created := false
	if childEntry == nil {
		nb.w.a++
		created = true
		// If we're not at the leaf yet, look ahead on the remaining path to determine what kind of intermediate parent
		// node we need to created.
		nextChildSeg, _ := remainingPath.Shift()
		if _, err = nextChildSeg.Index(); err == nil {
			// As per the discussion [here](https://github.com/smrz2001/go-ipld-prime/pull/1#issuecomment-1143035685),
			// this code assumes that if we're dealing with an integral path segment, it corresponds to a list index.
			childKind = datamodel.Kind_List
		} else {
			// From the same discussion as above, any non-integral, intermediate path can be assumed to be a map key.
			childKind = datamodel.Kind_Map
		}
	} else {
		childKind = childEntry.v.Build().Kind()
	}
	return nb.storeChildEntry(childKey, childEntry.v.Build(), childKind, created).Transform(remainingPath, transform, createParents)
}

func (nb *plainMap__Builder) storeChildEntry(k string, v datamodel.Node, kind datamodel.Kind, created bool) datamodel.NodeAmender {
	nb.w.m.Put(k, &plainMap__Entry{plainString(k), Prototype.Any.AmendingBuilder(v), created})
	return nb
}

// -- NodeAssembler -->

type plainMap__Assembler struct {
	w *plainMap

	ka plainMap__KeyAssembler
	va plainMap__ValueAssembler

	state maState
}
type plainMap__KeyAssembler struct {
	ma *plainMap__Assembler
}
type plainMap__ValueAssembler struct {
	ma *plainMap__Assembler
}

// maState is an enum of the state machine for a map assembler.
// (this might be something to export reusably, but it's also very much an impl detail that need not be seen, so, dubious.)
type maState uint8

const (
	maState_initial     maState = iota // also the 'expect key or finish' state
	maState_midKey                     // waiting for a 'finished' state in the KeyAssembler.
	maState_expectValue                // 'AssembleValue' is the only valid next step
	maState_midValue                   // waiting for a 'finished' state in the ValueAssembler.
	maState_finished                   // 'w' will also be nil, but this is a politer statement
)

func (na *plainMap__Assembler) BeginMap(sizeHint int64) (datamodel.MapAssembler, error) {
	if sizeHint < 0 {
		sizeHint = 0
	}
	// Allocate storage space.
	na.w.m = *linkedhashmap.New()
	// That's it; return self as the MapAssembler.  We already have all the right methods on this structure.
	return na, nil
}
func (plainMap__Assembler) BeginList(sizeHint int64) (datamodel.ListAssembler, error) {
	return mixins.MapAssembler{TypeName: "map"}.BeginList(0)
}
func (plainMap__Assembler) AssignNull() error {
	return mixins.MapAssembler{TypeName: "map"}.AssignNull()
}
func (plainMap__Assembler) AssignBool(bool) error {
	return mixins.MapAssembler{TypeName: "map"}.AssignBool(false)
}
func (plainMap__Assembler) AssignInt(int64) error {
	return mixins.MapAssembler{TypeName: "map"}.AssignInt(0)
}
func (plainMap__Assembler) AssignFloat(float64) error {
	return mixins.MapAssembler{TypeName: "map"}.AssignFloat(0)
}
func (plainMap__Assembler) AssignString(string) error {
	return mixins.MapAssembler{TypeName: "map"}.AssignString("")
}
func (plainMap__Assembler) AssignBytes([]byte) error {
	return mixins.MapAssembler{TypeName: "map"}.AssignBytes(nil)
}
func (plainMap__Assembler) AssignLink(datamodel.Link) error {
	return mixins.MapAssembler{TypeName: "map"}.AssignLink(nil)
}
func (na *plainMap__Assembler) AssignNode(v datamodel.Node) error {
	// Sanity check assembler state.
	//  Update of state to 'finished' comes later; where exactly depends on if shortcuts apply.
	if na.state != maState_initial {
		panic("misuse")
	}
	// Copy the content.
	if v2, ok := v.(*plainMap); ok { // if our own type: shortcut.
		// Copy the structure by value.
		//  This means we'll have pointers into the same internal maps and slices;
		//   this is okay, because the Node type promises it's immutable, and we are going to instantly finish ourselves to also maintain that.
		// FIXME: the shortcut behaves differently than the long way: it discards any existing progress.  Doesn't violate immut, but is odd.
		*na.w = *v2
		na.state = maState_finished
		return nil
	}
	// If the above shortcut didn't work, resort to a generic copy.
	//  We call AssignNode for all the child values, giving them a chance to hit shortcuts even if we didn't.
	if v.Kind() != datamodel.Kind_Map {
		return datamodel.ErrWrongKind{TypeName: "map", MethodName: "AssignNode", AppropriateKind: datamodel.KindSet_JustMap, ActualKind: v.Kind()}
	}
	itr := v.MapIterator()
	for !itr.Done() {
		k, v, err := itr.Next()
		if err != nil {
			return err
		}
		if err := na.AssembleKey().AssignNode(k); err != nil {
			return err
		}
		if err := na.AssembleValue().AssignNode(v); err != nil {
			return err
		}
	}
	return na.Finish()
}
func (plainMap__Assembler) Prototype() datamodel.NodePrototype {
	return Prototype.Map
}

// -- MapAssembler -->

// AssembleEntry is part of conforming to MapAssembler, which we do on
// plainMap__Assembler so that BeginMap can just return a retyped pointer rather than new object.
func (ma *plainMap__Assembler) AssembleEntry(k string) (datamodel.NodeAssembler, error) {
	// Sanity check assembler state.
	//  Update of state comes after possible key rejection.
	if ma.state != maState_initial {
		panic("misuse")
	}
	// Check for dup keys; error if so.
	if _, exists := ma.w.m.Get(k); exists {
		return nil, datamodel.ErrRepeatedMapKey{Key: plainString(k)}
	}
	ma.state = maState_midValue
	ma.w.m.Put(k, &plainMap__Entry{k: plainString(k)})
	// Make value assembler valid by giving it pointer back to whole 'ma'; yield it.
	ma.va.ma = ma
	return &ma.va, nil
}

// AssembleKey is part of conforming to MapAssembler, which we do on
// plainMap__Assembler so that BeginMap can just return a retyped pointer rather than new object.
func (ma *plainMap__Assembler) AssembleKey() datamodel.NodeAssembler {
	// Sanity check, then update, assembler state.
	if ma.state != maState_initial {
		panic("misuse")
	}
	ma.state = maState_midKey
	// Make key assembler valid by giving it pointer back to whole 'ma'; yield it.
	ma.ka.ma = ma
	return &ma.ka
}

// AssembleValue is part of conforming to MapAssembler, which we do on
// plainMap__Assembler so that BeginMap can just return a retyped pointer rather than new object.
func (ma *plainMap__Assembler) AssembleValue() datamodel.NodeAssembler {
	// Sanity check, then update, assembler state.
	if ma.state != maState_expectValue {
		panic("misuse")
	}
	ma.state = maState_midValue
	// Make value assembler valid by giving it pointer back to whole 'ma'; yield it.
	ma.va.ma = ma
	return &ma.va
}

// Finish is part of conforming to MapAssembler, which we do on
// plainMap__Assembler so that BeginMap can just return a retyped pointer rather than new object.
func (ma *plainMap__Assembler) Finish() error {
	// Sanity check, then update, assembler state.
	if ma.state != maState_initial {
		panic("misuse")
	}
	ma.state = maState_finished
	// validators could run and report errors promptly, if this type had any.
	return nil
}
func (plainMap__Assembler) KeyPrototype() datamodel.NodePrototype {
	return Prototype__String{}
}
func (plainMap__Assembler) ValuePrototype(_ string) datamodel.NodePrototype {
	return Prototype.Any
}

// -- MapAssembler.KeyAssembler -->

func (plainMap__KeyAssembler) BeginMap(sizeHint int64) (datamodel.MapAssembler, error) {
	return mixins.StringAssembler{TypeName: "string"}.BeginMap(0)
}
func (plainMap__KeyAssembler) BeginList(sizeHint int64) (datamodel.ListAssembler, error) {
	return mixins.StringAssembler{TypeName: "string"}.BeginList(0)
}
func (plainMap__KeyAssembler) AssignNull() error {
	return mixins.StringAssembler{TypeName: "string"}.AssignNull()
}
func (plainMap__KeyAssembler) AssignBool(bool) error {
	return mixins.StringAssembler{TypeName: "string"}.AssignBool(false)
}
func (plainMap__KeyAssembler) AssignInt(int64) error {
	return mixins.StringAssembler{TypeName: "string"}.AssignInt(0)
}
func (plainMap__KeyAssembler) AssignFloat(float64) error {
	return mixins.StringAssembler{TypeName: "string"}.AssignFloat(0)
}
func (mka *plainMap__KeyAssembler) AssignString(v string) error {
	// Check for dup keys; error if so.
	//  (And, backtrack state to accepting keys again so we don't get eternally wedged here.)
	if _, exists := mka.ma.w.m.Get(v); exists {
		mka.ma.state = maState_initial
		mka.ma = nil // invalidate self to prevent further incorrect use.
		return datamodel.ErrRepeatedMapKey{Key: plainString(v)}
	}
	// Assign the key into the end of the entry table;
	//  we'll be doing map insertions after we get the value in hand.
	//  (There's no need to delegate to another assembler for the key type,
	//   because we're just at Data Model level here, which only regards plain strings.)
	mka.ma.w.m.Put(v, &plainMap__Entry{k: plainString(v)})
	// Update parent assembler state: clear to proceed.
	mka.ma.state = maState_expectValue
	mka.ma = nil // invalidate self to prevent further incorrect use.
	return nil
}
func (plainMap__KeyAssembler) AssignBytes([]byte) error {
	return mixins.StringAssembler{TypeName: "string"}.AssignBytes(nil)
}
func (plainMap__KeyAssembler) AssignLink(datamodel.Link) error {
	return mixins.StringAssembler{TypeName: "string"}.AssignLink(nil)
}
func (mka *plainMap__KeyAssembler) AssignNode(v datamodel.Node) error {
	vs, err := v.AsString()
	if err != nil {
		return fmt.Errorf("cannot assign non-string node into map key assembler") // FIXME:errors: this doesn't quite fit in ErrWrongKind cleanly; new error type?
	}
	return mka.AssignString(vs)
}
func (plainMap__KeyAssembler) Prototype() datamodel.NodePrototype {
	return Prototype__String{}
}

// -- MapAssembler.ValueAssembler -->

func (mva *plainMap__ValueAssembler) BeginMap(sizeHint int64) (datamodel.MapAssembler, error) {
	ma := plainMap__ValueAssemblerMap{}
	ma.ca.w = &plainMap{}
	ma.p = mva.ma
	_, err := ma.ca.BeginMap(sizeHint)
	return &ma, err
}
func (mva *plainMap__ValueAssembler) BeginList(sizeHint int64) (datamodel.ListAssembler, error) {
	la := plainMap__ValueAssemblerList{}
	la.ca.w = &plainList{}
	la.p = mva.ma
	_, err := la.ca.BeginList(sizeHint)
	return &la, err
}
func (mva *plainMap__ValueAssembler) AssignNull() error {
	return mva.AssignNode(datamodel.Null)
}
func (mva *plainMap__ValueAssembler) AssignBool(v bool) error {
	vb := plainBool(v)
	return mva.AssignNode(&vb)
}
func (mva *plainMap__ValueAssembler) AssignInt(v int64) error {
	vb := plainInt(v)
	return mva.AssignNode(&vb)
}
func (mva *plainMap__ValueAssembler) AssignFloat(v float64) error {
	vb := plainFloat(v)
	return mva.AssignNode(&vb)
}
func (mva *plainMap__ValueAssembler) AssignString(v string) error {
	vb := plainString(v)
	return mva.AssignNode(&vb)
}
func (mva *plainMap__ValueAssembler) AssignBytes(v []byte) error {
	vb := plainBytes(v)
	return mva.AssignNode(&vb)
}
func (mva *plainMap__ValueAssembler) AssignLink(v datamodel.Link) error {
	vb := plainLink{v}
	return mva.AssignNode(&vb)
}
func (mva *plainMap__ValueAssembler) AssignNode(v datamodel.Node) error {
	itr := mva.ma.w.m.Iterator()
	itr.Last()
	val := itr.Value().(*plainMap__Entry)
	nb := Prototype.Any.AmendingBuilder(v)
	val.v = nb
	val.a = true
	mva.ma.w.a++
	mva.ma.state = maState_initial
	mva.ma = nil // invalidate self to prevent further incorrect use.
	return nil
}
func (plainMap__ValueAssembler) Prototype() datamodel.NodePrototype {
	return Prototype.Any
}

type plainMap__ValueAssemblerMap struct {
	ca plainMap__Assembler
	p  *plainMap__Assembler // pointer back to parent, for final insert and state bump
}

// we briefly state only the methods we need to delegate here.
// just embedding plainMap__Assembler also behaves correctly,
//  but causes a lot of unnecessary autogenerated functions in the final binary.

func (ma *plainMap__ValueAssemblerMap) AssembleEntry(k string) (datamodel.NodeAssembler, error) {
	return ma.ca.AssembleEntry(k)
}
func (ma *plainMap__ValueAssemblerMap) AssembleKey() datamodel.NodeAssembler {
	return ma.ca.AssembleKey()
}
func (ma *plainMap__ValueAssemblerMap) AssembleValue() datamodel.NodeAssembler {
	return ma.ca.AssembleValue()
}
func (plainMap__ValueAssemblerMap) KeyPrototype() datamodel.NodePrototype {
	return Prototype__String{}
}
func (plainMap__ValueAssemblerMap) ValuePrototype(_ string) datamodel.NodePrototype {
	return Prototype.Any
}

func (ma *plainMap__ValueAssemblerMap) Finish() error {
	if err := ma.ca.Finish(); err != nil {
		return err
	}
	w := ma.ca.w
	ma.ca.w = nil
	return ma.p.va.AssignNode(w)
}

type plainMap__ValueAssemblerList struct {
	ca plainList__Assembler
	p  *plainMap__Assembler // pointer back to parent, for final insert and state bump
}

// we briefly state only the methods we need to delegate here.
// just embedding plainList__Assembler also behaves correctly,
//  but causes a lot of unnecessary autogenerated functions in the final binary.

func (la *plainMap__ValueAssemblerList) AssembleValue() datamodel.NodeAssembler {
	return la.ca.AssembleValue()
}
func (plainMap__ValueAssemblerList) ValuePrototype(_ int64) datamodel.NodePrototype {
	return Prototype.Any
}

func (la *plainMap__ValueAssemblerList) Finish() error {
	if err := la.ca.Finish(); err != nil {
		return err
	}
	w := la.ca.w
	la.ca.w = nil
	return la.p.va.AssignNode(w)
}
