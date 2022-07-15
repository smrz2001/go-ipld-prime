package traversal

import (
	"fmt"
	"github.com/emirpasic/gods/maps/linkedhashmap"

	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/ipld/go-ipld-prime/node/mixins"
)

var (
	_ datamodel.Node = &mapAmender{}
	_ Amender        = &mapAmender{}
)

type mapAmender struct {
	cfg     *AmendOptions
	base    datamodel.Node
	parent  Amender
	created bool
	// This is the information needed to present an accurate "effective" view of the base node and all accumulated
	// modifications.
	mods linkedhashmap.Map
	// This is the count of children *present in the base node* that are removed. Knowing this count allows accurate
	// traversal of the "effective" node view.
	rems int
	// This is the count of new children. If an added node is removed, this count should be decremented instead of
	// `rems`.
	adds int
}

func (cfg AmendOptions) newMapAmender(base datamodel.Node, parent Amender, create bool) Amender {
	// If the base node is already a map-amender, reuse the mutation state but reset `parent` and `created`.
	if amd, castOk := base.(*mapAmender); castOk {
		return &mapAmender{&cfg, amd.base, parent, create, amd.mods, amd.rems, amd.adds}
	} else {
		// Start with fresh state because existing metadata could not be reused.
		return &mapAmender{&cfg, base, parent, create, *linkedhashmap.New(), 0, 0}
	}
}

// -- Node -->

func (a *mapAmender) Kind() datamodel.Kind {
	return datamodel.Kind_Map
}

func (a *mapAmender) LookupByString(key string) (datamodel.Node, error) {
	seg := datamodel.PathSegmentOfString(key)
	// Added/removed nodes override the contents of the base node
	if mod, exists := a.mods.Get(seg); exists {
		v := mod.(Amender).Build()
		if v.IsNull() {
			return nil, datamodel.ErrNotExists{Segment: seg}
		}
		return v, nil
	}
	// Fallback to base node
	if a.base != nil {
		return a.base.LookupByString(key)
	}
	return nil, datamodel.ErrNotExists{Segment: seg}
}

func (a *mapAmender) LookupByNode(key datamodel.Node) (datamodel.Node, error) {
	ks, err := key.AsString()
	if err != nil {
		return nil, err
	}
	return a.LookupByString(ks)
}

func (a *mapAmender) LookupByIndex(idx int64) (datamodel.Node, error) {
	return mixins.Map{TypeName: "mapAmender"}.LookupByIndex(idx)
}

func (a *mapAmender) LookupBySegment(seg datamodel.PathSegment) (datamodel.Node, error) {
	return a.LookupByString(seg.String())
}

func (a *mapAmender) MapIterator() datamodel.MapIterator {
	var baseItr datamodel.MapIterator = nil
	// If all children were removed from the base node, or no base node was specified, there is nothing to iterate
	// over w.r.t. that node.
	if (a.base != nil) && (int64(a.rems) < a.base.Length()) {
		baseItr = a.base.MapIterator()
	}
	var modsItr *linkedhashmap.Iterator
	if (a.rems != 0) || (a.adds != 0) {
		itr := a.mods.Iterator()
		modsItr = &itr
	}
	return &mapAmender_Iterator{a, modsItr, baseItr, 0}
}

func (a *mapAmender) ListIterator() datamodel.ListIterator {
	return nil
}

func (a *mapAmender) Length() int64 {
	length := int64(a.adds - a.rems)
	if a.base != nil {
		length = length + a.base.Length()
	}
	return length
}

func (a *mapAmender) IsAbsent() bool {
	return false
}

func (a *mapAmender) IsNull() bool {
	return false
}

func (a *mapAmender) AsBool() (bool, error) {
	return mixins.Map{TypeName: "mapAmender"}.AsBool()
}

func (a *mapAmender) AsInt() (int64, error) {
	return mixins.Map{TypeName: "mapAmender"}.AsInt()
}

func (a *mapAmender) AsFloat() (float64, error) {
	return mixins.Map{TypeName: "mapAmender"}.AsFloat()
}

func (a *mapAmender) AsString() (string, error) {
	return mixins.Map{TypeName: "mapAmender"}.AsString()
}

func (a *mapAmender) AsBytes() ([]byte, error) {
	return mixins.Map{TypeName: "mapAmender"}.AsBytes()
}

func (a *mapAmender) AsLink() (datamodel.Link, error) {
	return mixins.Map{TypeName: "mapAmender"}.AsLink()
}

func (a *mapAmender) Prototype() datamodel.NodePrototype {
	return basicnode.Prototype.Map
}

type mapAmender_Iterator struct {
	amd     *mapAmender
	modsItr *linkedhashmap.Iterator
	baseItr datamodel.MapIterator
	idx     int
}

func (itr *mapAmender_Iterator) Next() (k datamodel.Node, v datamodel.Node, _ error) {
	if itr.Done() {
		return nil, nil, datamodel.ErrIteratorOverread{}
	}
	if itr.baseItr != nil {
		// Iterate over base node first to maintain ordering.
		var err error
		for !itr.baseItr.Done() {
			k, v, err = itr.baseItr.Next()
			if err != nil {
				return nil, nil, err
			}
			ks, _ := k.AsString()
			if err != nil {
				return nil, nil, err
			}
			if mod, exists := itr.amd.mods.Get(datamodel.PathSegmentOfString(ks)); exists {
				v = mod.(Amender).Build()
				// Skip removed nodes
				if v.IsNull() {
					continue
				}
				// Fall-through and return wrapped nodes
			}
			// We found a "real" node to return, increment the counter.
			itr.idx++
			return
		}
	}
	if itr.modsItr != nil {
		// Iterate over mods, skipping removed nodes.
		for itr.modsItr.Next() {
			key := itr.modsItr.Key()
			k = basicnode.NewString(key.(datamodel.PathSegment).String())
			v = itr.modsItr.Value().(Amender).Build()
			// Skip removed nodes.
			if v.IsNull() {
				continue
			}
			// Skip "wrapper" nodes that represent existing sub-nodes in the hierarchy corresponding to an added leaf
			// node.
			if !isCreated(v) {
				continue
			}
			// We found a "real" node to return, increment the counter.
			itr.idx++
			return
		}
	}
	return nil, nil, datamodel.ErrIteratorOverread{}
}

func (itr *mapAmender_Iterator) Done() bool {
	// Iteration is complete when all source nodes have been processed (skipping removed nodes) and all mods have been
	// processed.
	return int64(itr.idx) >= itr.amd.Length()
}

// -- Amender -->

func (a *mapAmender) Get(prog *Progress, path datamodel.Path, trackProgress bool) (datamodel.Node, error) {
	// If the root is requested, return the `Node` view of the amender.
	if path.Len() == 0 {
		return a.Build(), nil
	}
	// Check the budget
	if prog.Budget != nil {
		if prog.Budget.NodeBudget <= 0 {
			return nil, &ErrBudgetExceeded{BudgetKind: "node", Path: prog.Path}
		}
		prog.Budget.NodeBudget--
	}
	childSeg, remainingPath := path.Shift()
	prog.Path = prog.Path.AppendSegment(childSeg)
	childVal, err := a.LookupBySegment(childSeg)
	// Since we're explicitly looking for a node, look for the child node in the current amender state and throw an
	// error if it does not exist.
	if err != nil {
		return nil, err
	}
	return a.storeChildAmender(childSeg, childVal, childVal.Kind(), false, trackProgress).Get(prog, remainingPath, trackProgress)
}

func (a *mapAmender) Transform(prog *Progress, path datamodel.Path, fn TransformFn, createParents bool) (datamodel.Node, error) {
	// Allow the base node to be replaced.
	if path.Len() == 0 {
		prevNode := a.Build()
		if newNode, err := fn(*prog, prevNode); err != nil {
			return nil, err
		} else if newNode.Kind() != datamodel.Kind_Map {
			return nil, fmt.Errorf("transform: cannot transform root into incompatible type: %q", newNode.Kind())
		} else {
			// Go through `newMapAmender` in case `newNode` is already a map-amender.
			*a = *a.cfg.newMapAmender(newNode, a.parent, a.created).(*mapAmender)
			return prevNode, nil
		}
	}
	// Check the budget
	if prog.Budget != nil {
		if prog.Budget.NodeBudget <= 0 {
			return nil, &ErrBudgetExceeded{BudgetKind: "node", Path: prog.Path}
		}
		prog.Budget.NodeBudget--
	}
	childSeg, remainingPath := path.Shift()
	prog.Path = prog.Path.AppendSegment(childSeg)
	atLeaf := remainingPath.Len() == 0
	childVal, err := a.LookupBySegment(childSeg)
	if err != nil {
		// - Return any error other than "not exists".
		// - If the child node does not exist and `createParents = true`, create the new hierarchy, otherwise throw an
		//   error.
		// - Even if `createParent = false`, if we're at the leaf, don't throw an error because we don't need to create
		//   any more intermediate parent nodes.
		if _, notFoundErr := err.(datamodel.ErrNotExists); !notFoundErr || !(atLeaf || createParents) {
			return nil, fmt.Errorf("transform: parent position at %q did not exist (and createParents was false)", prog.Path)
		}
	}
	if atLeaf {
		if newChildVal, err := fn(*prog, childVal); err != nil {
			return nil, err
		} else if newChildVal == nil {
			// Use the "Null" node to indicate a removed child.
			a.mods.Put(childSeg, a.cfg.newAnyAmender(datamodel.Null, a, false))
			// If the child node being removed is a new node previously added to the node hierarchy, decrement `adds`,
			// otherwise increment `rems`. This allows us to retain knowledge about the "history" of the base hierarchy.
			if isCreated(newChildVal) {
				a.rems++
			} else {
				a.adds--
			}
		} else {
			// While building the nested amender tree, only count nodes as "added" when they didn't exist and had to be
			// created to fill out the hierarchy.
			create := false
			if childVal == nil {
				a.adds++
				create = true
			}
			a.storeChildAmender(childSeg, newChildVal, newChildVal.Kind(), create, true)
		}
		return childVal, nil
	}
	// While building the nested amender tree, only count nodes as "added" when they didn't exist and had to be created
	// to fill out the hierarchy.
	var childKind datamodel.Kind
	create := false
	if childVal == nil {
		a.adds++
		create = true
		// If we're not at the leaf yet, look ahead on the remaining path to determine what kind of intermediate parent
		// node we need to create.
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
		childKind = childVal.Kind()
	}
	return a.storeChildAmender(childSeg, childVal, childKind, create, true).Transform(prog, remainingPath, fn, createParents)
}

func (a *mapAmender) Build() datamodel.Node {
	// `mapAmender` is also a `Node`.
	return (datamodel.Node)(a)
}

func (a *mapAmender) storeChildAmender(seg datamodel.PathSegment, n datamodel.Node, k datamodel.Kind, create bool, trackProgress bool) Amender {
	childAmender := a.cfg.newAmender(n, a, k, create)
	if trackProgress {
		a.mods.Put(seg, childAmender)
	}
	return childAmender
}
