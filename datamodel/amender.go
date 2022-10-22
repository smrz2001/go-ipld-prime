package datamodel

type NodeAmender interface {
	NodeBuilder

	// Get returns the node at the specified path. It will not create any intermediate nodes because this is just a
	// retrieval and not a modification operation.
	Get(cfg NodeAmendCfg, path Path) (Node, error)

	// Transform will do an in-place transformation of the node at the specified path and return its previous value.
	// If `createParents = true`, any missing parents will be created, otherwise this function will return an error.
	Transform(cfg NodeAmendCfg, path Path, transform func(Node) (Node, error), createParents bool) (Node, error)
}

type NodeAmendCfg struct {
	LinkLoader func(Link) (Node, error)
	LinkStorer func(LinkPrototype, Node) (Link, error)
}
