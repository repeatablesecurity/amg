package execution

// TypeOfNode denotes the type of node playbook
type TypeOfNode string

const (
	// ActionNodeT denotes node that has an action to execute
	ActionNodeT = "ActionNode"
	// IfNodeT denotes If condition
	IfNodeT = "IfNode"
	// ForNodeT denotes For loop
	ForNodeT = "ForNode"
)

// Node defines basic node level operations like Next() etc
type Node interface {
	Next() Node
	SetNext(Node)
	NodeType() TypeOfNode
}

// GenericExecutionNode defines basic data members to support node level operations.
type GenericExecutionNode struct {
	Id         string
	typeOfNode TypeOfNode
	nextNode   Node
}

func (n *GenericExecutionNode) Next() Node {
	return n.nextNode
}

func (n *GenericExecutionNode) NodeType() TypeOfNode {
	return n.typeOfNode
}

func (n *GenericExecutionNode) SetNext(next Node) {
	n.nextNode = next
}

type ActionNode struct {
	GenericExecutionNode
	urn                 string
	inputParams         map[string]string
	concreteInputParams map[string]string
	// varName -> expression(result)
	//	or simply, varName -> [result | result.fieldName ]
	exportResultAs map[string]string
}

// May be remove this.
func NewActionNode(urn string, ip map[string]string) *ActionNode {
	return &ActionNode{
		GenericExecutionNode: GenericExecutionNode{"", ActionNodeT, nil},
		urn:                  urn,
		inputParams:          ip,
	}
}

type IfNode struct {
	GenericExecutionNode
	condition        string
	YesPathFirstNode Node
	NoPathFirstNode  Node
}

type ForNode struct {
	GenericExecutionNode
	IterateOnVar  string
	FirstLoopNode Node
}
