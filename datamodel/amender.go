package datamodel

type NodeAmender interface {
	NodeBuilder

	// Get returns the node at the specified path. It will not create any intermediate nodes because this is just a
	// retrieval and not a modification operation.
	Get(path Path) (Node, error)

	// Transform will do an in-place transformation of the node at the specified path and return its previous value.
	// If `createParents = true`, any missing parents will be created, otherwise this function will return an error.
	Transform(path Path, createParents bool) (Node, error)

	// Amend returns a `Node` representing the "effective" view of the base `Node` (if specified) along with accumulated
	// update (if any).
	Amend() Node
}
