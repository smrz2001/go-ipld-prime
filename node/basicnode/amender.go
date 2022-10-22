package basicnode

import "github.com/ipld/go-ipld-prime/datamodel"

func NewAmender(base datamodel.Node, kind datamodel.Kind) datamodel.NodeAmender {
	if kind == datamodel.Kind_Map {
		return Prototype.Map.AmendingBuilder(base)
	} else if kind == datamodel.Kind_List {
		return Prototype.List.AmendingBuilder(base)
	} else if kind == datamodel.Kind_Link {
		return Prototype.Link.AmendingBuilder(base)
	} else {
		return Prototype.Any.AmendingBuilder(base)
	}
}
