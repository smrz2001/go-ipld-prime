package basicnode

import "github.com/ipld/go-ipld-prime/datamodel"

func NewAmender(base datamodel.Node, kind datamodel.Kind) datamodel.NodeAmender {
	if kind == datamodel.Kind_Map {
		return Prototype.Map.amender(base)
	} else if kind == datamodel.Kind_List {
		return Prototype.List.amender(base)
	} else {
		return Prototype.Any.amender(base)
	}
}
