package healing

import (
	"context"
	"fmt"

	"github.com/alexandrmotologa/domdrift/internal/browser"
)

// ResolvedNode encapsulates the node identity and hierarchy for selector synthesis.
type ResolvedNode struct {
	NodeID        int64
	BackendNodeID int64
	Node          *browser.DOMNode
	X             int
	Y             int
}

// NodeResolver maps pixel coordinates to Chrome DOM nodes.
type NodeResolver struct{}

// NewNodeResolver creates a coordinate resolver.
func NewNodeResolver() *NodeResolver {
	return &NodeResolver{}
}

// Resolve maps viewport coordinates (x, y) to a DOM node and extracts its tree hierarchy.
func (r *NodeResolver) Resolve(ctx context.Context, sess *browser.Session, x, y int) (*ResolvedNode, error) {
	nodeID, backendNodeID, err := sess.GetNodeForLocation(ctx, x, y)
	if err != nil {
		return nil, fmt.Errorf("failed locating node at (%d, %d): %w", x, y, err)
	}

	if nodeID == 0 && backendNodeID == 0 {
		return nil, fmt.Errorf("no DOM node detected at coordinates (%d, %d)", x, y)
	}

	var node *browser.DOMNode
	if nodeID > 0 {
		node, err = sess.DescribeNode(ctx, nodeID, 5)
	} else if backendNodeID > 0 {
		node, err = sess.DescribeBackendNode(ctx, backendNodeID, 5)
	}

	if err != nil {
		return nil, fmt.Errorf("failed describing resolved node: %w", err)
	}

	return &ResolvedNode{
		NodeID:        nodeID,
		BackendNodeID: backendNodeID,
		Node:          node,
		X:             x,
		Y:             y,
	}, nil
}
