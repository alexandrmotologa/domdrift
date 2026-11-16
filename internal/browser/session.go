package browser

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// DOMNode represents a node in the Chrome DOM tree.
type DOMNode struct {
	NodeID           int64              `json:"nodeId"`
	BackendNodeID    int64              `json:"backendNodeId"`
	NodeType         int                `json:"nodeType"`
	NodeName         string             `json:"nodeName"`
	LocalName        string             `json:"localName"`
	NodeValue        string             `json:"nodeValue"`
	ParentID         int64              `json:"parentId,omitempty"`
	ChildNodeCount   int                `json:"childNodeCount,omitempty"`
	Children         []*DOMNode         `json:"children,omitempty"`
	Attributes       []string           `json:"attributes,omitempty"`
	DocumentURL      string             `json:"documentURL,omitempty"`
	BaseURL          string             `json:"baseURL,omitempty"`
	ShadowRoots      []*DOMNode         `json:"shadowRoots,omitempty"`
	AttributeMapCache map[string]string `json:"-"`
}

// GetAttribute returns the value of an attribute by name.
func (n *DOMNode) GetAttribute(name string) string {
	if n == nil {
		return ""
	}
	if n.AttributeMapCache == nil {
		n.AttributeMapCache = make(map[string]string)
		for i := 0; i+1 < len(n.Attributes); i += 2 {
			n.AttributeMapCache[n.Attributes[i]] = n.Attributes[i+1]
		}
	}
	return n.AttributeMapCache[name]
}

// AXValue represents an accessible attribute value.
type AXValue struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

// AXNode represents a Chrome Accessibility Tree node.
type AXNode struct {
	NodeID           string     `json:"nodeId"`
	Ignored          bool       `json:"ignored"`
	Role             *AXValue   `json:"role,omitempty"`
	Name             *AXValue   `json:"name,omitempty"`
	Description      *AXValue   `json:"description,omitempty"`
	Value            *AXValue   `json:"value,omitempty"`
	BackendDOMNodeID int64      `json:"backendDOMNodeId,omitempty"`
	ChildIDs         []string   `json:"childIds,omitempty"`
	ParentID         string     `json:"parentId,omitempty"`
}

// RoleName returns string value of node role.
func (a *AXNode) RoleName() string {
	if a.Role != nil && a.Role.Value != nil {
		return fmt.Sprintf("%v", a.Role.Value)
	}
	return ""
}

// NameString returns string value of accessible name.
func (a *AXNode) NameString() string {
	if a.Name != nil && a.Name.Value != nil {
		return fmt.Sprintf("%v", a.Name.Value)
	}
	return ""
}

// BoxModel represents element layout dimensions.
type BoxModel struct {
	Content []float64 `json:"content"`
	Padding []float64 `json:"padding"`
	Border  []float64 `json:"border"`
	Margin  []float64 `json:"margin"`
	Width   int       `json:"width"`
	Height  int       `json:"height"`
}

// Session manages a dedicated browser tab session.
type Session struct {
	client    *CDPClient
	targetID  string
	sessionID string

	mu           sync.Mutex
	loadedCh     chan struct{}
	lastURL      string
	isAttached   bool
}

// NewSession creates and attaches to a new browser target.
func NewSession(ctx context.Context, client *CDPClient) (*Session, error) {
	// Create blank target
	res, err := client.Call(ctx, "Target.createTarget", map[string]interface{}{
		"url": "about:blank",
	})
	if err != nil {
		return nil, fmt.Errorf("failed creating target: %w", err)
	}

	var createTargetRes struct {
		TargetID string `json:"targetId"`
	}
	if err := json.Unmarshal(res, &createTargetRes); err != nil {
		return nil, fmt.Errorf("invalid createTarget response: %w", err)
	}

	// Attach to target
	attachRes, err := client.Call(ctx, "Target.attachToTarget", map[string]interface{}{
		"targetId": createTargetRes.TargetID,
		"flatten":  true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed attaching to target: %w", err)
	}

	var attachTargetRes struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.Unmarshal(attachRes, &attachTargetRes); err != nil {
		return nil, fmt.Errorf("invalid attachToTarget response: %w", err)
	}

	sess := &Session{
		client:     client,
		targetID:   createTargetRes.TargetID,
		sessionID:  attachTargetRes.SessionID,
		loadedCh:   make(chan struct{}, 1),
		isAttached: true,
	}

	// Listen for page events
	client.OnEvent(func(method string, params json.RawMessage) {
		if method == "Page.loadEventFired" {
			sess.mu.Lock()
			select {
			case sess.loadedCh <- struct{}{}:
			default:
			}
			sess.mu.Unlock()
		}
	})

	// Enable standard domains
	domains := []string{"Page.enable", "Runtime.enable", "DOM.enable", "Accessibility.enable"}
	for _, d := range domains {
		if _, err := sess.call(ctx, d, nil); err != nil {
			_ = sess.Close(ctx)
			return nil, fmt.Errorf("failed enabling %s: %w", d, err)
		}
	}

	// Register stealth evasions script on every new document
	_ = sess.ApplyStealthEvasions(ctx)

	return sess, nil
}

func (s *Session) call(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	return s.client.CallWithSession(ctx, s.sessionID, method, params)
}

// TargetID returns the target ID of this tab.
func (s *Session) TargetID() string {
	return s.targetID
}

// SessionID returns the CDP session ID.
func (s *Session) SessionID() string {
	return s.sessionID
}

// Navigate points the page to the target URL and waits for page load.
func (s *Session) Navigate(ctx context.Context, url string) error {
	s.mu.Lock()
	// Drain any existing load signals
	select {
	case <-s.loadedCh:
	default:
	}
	s.lastURL = url
	s.mu.Unlock()

	_, err := s.call(ctx, "Page.navigate", map[string]interface{}{
		"url": url,
	})
	if err != nil {
		return fmt.Errorf("navigate to %s failed: %w", url, err)
	}

	// Wait for loadEventFired with timeout or context cancel
	select {
	case <-s.loadedCh:
		// Brief pause for script hydration
		time.Sleep(200 * time.Millisecond)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(20 * time.Second):
		// Even if loadEventFired doesn't fire (e.g. infinite polling), continue
		return nil
	}
}

// Evaluate runs a JavaScript expression in the page context and returns the result.
func (s *Session) Evaluate(ctx context.Context, expr string) (json.RawMessage, error) {
	res, err := s.call(ctx, "Runtime.evaluate", map[string]interface{}{
		"expression":    expr,
		"returnByValue": true,
		"awaitPromise":  true,
	})
	if err != nil {
		return nil, err
	}

	var evalRes struct {
		Result struct {
			Type  string          `json:"type"`
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		ExceptionDetails *struct {
			Text string `json:"text"`
		} `json:"exceptionDetails,omitempty"`
	}

	if err := json.Unmarshal(res, &evalRes); err != nil {
		return nil, fmt.Errorf("failed unmarshaling evaluate response: %w", err)
	}

	if evalRes.ExceptionDetails != nil {
		return nil, fmt.Errorf("javascript evaluation exception: %s", evalRes.ExceptionDetails.Text)
	}

	return evalRes.Result.Value, nil
}

// CaptureScreenshot takes a viewport or full-page screenshot.
func (s *Session) CaptureScreenshot(ctx context.Context, format string) ([]byte, error) {
	if format == "" {
		format = "png"
	}

	params := map[string]interface{}{
		"format":  format,
		"quality": 85,
	}

	res, err := s.call(ctx, "Page.captureScreenshot", params)
	if err != nil {
		return nil, fmt.Errorf("captureScreenshot failed: %w", err)
	}

	var captureRes struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(res, &captureRes); err != nil {
		return nil, fmt.Errorf("failed unmarshaling screenshot data: %w", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(captureRes.Data)
	if err != nil {
		return nil, fmt.Errorf("failed decoding base64 screenshot: %w", err)
	}

	return decoded, nil
}

// GetDocument returns the root DOM document tree.
func (s *Session) GetDocument(ctx context.Context, depth int) (*DOMNode, error) {
	if depth <= 0 {
		depth = -1
	}

	res, err := s.call(ctx, "DOM.getDocument", map[string]interface{}{
		"depth": depth,
		"pierce": true,
	})
	if err != nil {
		return nil, fmt.Errorf("getDocument failed: %w", err)
	}

	var docRes struct {
		Root *DOMNode `json:"root"`
	}
	if err := json.Unmarshal(res, &docRes); err != nil {
		return nil, fmt.Errorf("failed unmarshaling root DOM node: %w", err)
	}

	return docRes.Root, nil
}

// QuerySelector returns the node ID for a given CSS selector within a parent node.
func (s *Session) QuerySelector(ctx context.Context, parentNodeID int64, selector string) (int64, error) {
	res, err := s.call(ctx, "DOM.querySelector", map[string]interface{}{
		"nodeId":   parentNodeID,
		"selector": selector,
	})
	if err != nil {
		return 0, err
	}

	var queryRes struct {
		NodeID int64 `json:"nodeId"`
	}
	if err := json.Unmarshal(res, &queryRes); err != nil {
		return 0, err
	}

	return queryRes.NodeID, nil
}

// QuerySelectorAll returns all node IDs matching a selector.
func (s *Session) QuerySelectorAll(ctx context.Context, parentNodeID int64, selector string) ([]int64, error) {
	res, err := s.call(ctx, "DOM.querySelectorAll", map[string]interface{}{
		"nodeId":   parentNodeID,
		"selector": selector,
	})
	if err != nil {
		return nil, err
	}

	var queryRes struct {
		NodeIDs []int64 `json:"nodeIds"`
	}
	if err := json.Unmarshal(res, &queryRes); err != nil {
		return nil, err
	}

	return queryRes.NodeIDs, nil
}

// GetNodeForLocation maps (x, y) coordinates to a DOM node ID.
func (s *Session) GetNodeForLocation(ctx context.Context, x, y int) (int64, int64, error) {
	res, err := s.call(ctx, "DOM.getNodeForLocation", map[string]interface{}{
		"x":                          x,
		"y":                          y,
		"includeUserAgentShadowDOM": true,
		"ignorePointerEventsNone":   true,
	})
	if err != nil {
		return 0, 0, fmt.Errorf("getNodeForLocation(%d, %d) failed: %w", x, y, err)
	}

	var locRes struct {
		NodeID        int64 `json:"nodeId"`
		BackendNodeID int64 `json:"backendNodeId"`
	}
	if err := json.Unmarshal(res, &locRes); err != nil {
		return 0, 0, err
	}

	return locRes.NodeID, locRes.BackendNodeID, nil
}

// DescribeNode returns detailed DOM attributes and tags for a node ID.
func (s *Session) DescribeNode(ctx context.Context, nodeID int64, depth int) (*DOMNode, error) {
	if depth <= 0 {
		depth = 1
	}

	res, err := s.call(ctx, "DOM.describeNode", map[string]interface{}{
		"nodeId": nodeID,
		"depth":  depth,
		"pierce": true,
	})
	if err != nil {
		return nil, fmt.Errorf("describeNode(%d) failed: %w", nodeID, err)
	}

	var descRes struct {
		Node *DOMNode `json:"node"`
	}
	if err := json.Unmarshal(res, &descRes); err != nil {
		return nil, err
	}

	return descRes.Node, nil
}

// DescribeBackendNode resolves a backend node ID into DOM node details.
func (s *Session) DescribeBackendNode(ctx context.Context, backendNodeID int64, depth int) (*DOMNode, error) {
	if depth <= 0 {
		depth = 1
	}

	res, err := s.call(ctx, "DOM.describeNode", map[string]interface{}{
		"backendNodeId": backendNodeID,
		"depth":         depth,
		"pierce":        true,
	})
	if err != nil {
		return nil, fmt.Errorf("describeNode(backendNodeId=%d) failed: %w", backendNodeID, err)
	}

	var descRes struct {
		Node *DOMNode `json:"node"`
	}
	if err := json.Unmarshal(res, &descRes); err != nil {
		return nil, err
	}

	return descRes.Node, nil
}

// GetBoxModel returns bounding box quads for a node.
func (s *Session) GetBoxModel(ctx context.Context, nodeID int64) (*BoxModel, error) {
	res, err := s.call(ctx, "DOM.getBoxModel", map[string]interface{}{
		"nodeId": nodeID,
	})
	if err != nil {
		return nil, err
	}

	var boxRes struct {
		Model *BoxModel `json:"model"`
	}
	if err := json.Unmarshal(res, &boxRes); err != nil {
		return nil, err
	}

	return boxRes.Model, nil
}

// GetAccessibilityTree retrieves the full Chrome accessibility tree.
func (s *Session) GetAccessibilityTree(ctx context.Context) ([]AXNode, error) {
	res, err := s.call(ctx, "Accessibility.getFullAXTree", map[string]interface{}{
		"depth": -1,
	})
	if err != nil {
		return nil, fmt.Errorf("getFullAXTree failed: %w", err)
	}

	var axRes struct {
		Nodes []AXNode `json:"nodes"`
	}
	if err := json.Unmarshal(res, &axRes); err != nil {
		return nil, err
	}

	return axRes.Nodes, nil
}

// Close closes the target tab and detaches session.
func (s *Session) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isAttached {
		return nil
	}
	s.isAttached = false

	_, err := s.client.Call(ctx, "Target.closeTarget", map[string]interface{}{
		"targetId": s.targetID,
	})
	return err
}
