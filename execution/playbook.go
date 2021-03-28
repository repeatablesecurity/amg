package execution

import (
	"fmt"
	"strconv"

	"gopkg.in/yaml.v2" // https://github.com/go-yaml/yaml
)

var count int = 1

// type N struct {
// 	// Common fields
// 	ID       string `yaml:"id"`
// 	NodeType string `yaml:"type"`
// 	// Fields for Action node
// 	Urn     string
// 	Params  map[string]string
// 	Exports map[string]string
// 	// Fields for IF node
// 	Condition string
// 	OnTrue    []N `yaml:"onTrue,flow"`
// 	OnFalse   []N `yaml:"onFalse,flow"`
// }

type N struct {
	ID             string            `yaml:"id,omitempty" json:"id,omitempty"`
	NodeType       string            `yaml:"type,omitempty" json:"type,omitempty"`
	State          string            `yaml:"state,omitempty" json:"state,omitempty"` // Possible values are done|waitingVarResolution|waitingForResult|notYetStarted
	Err            string            `yaml:"error,omitempty" json:"error,omitempty"`
	Unresolved     []string          `yaml:"varsUnresolved,omitempty" json:"varsUnresolved,omitempty"`
	ResolvedValues map[string]string `yaml:"varsResolvedValues,omitempty" json:"varsResolvedValues,omitempty"`
	// Result for Action node
	Urn                 string            `yaml:",omitempty" json:",omitempty"`
	Params              map[string]string `yaml:",omitempty" json:",omitempty"`
	ConcreteInputParams map[string]string `yaml:"concreteParams,omitempty" json:"concreteParams,omitempty"`
	ResultFields        map[string]string `yaml:"resultFields,omitempty" json:"resultFields,omitempty"`
	RawResult           string            `yaml:"resultRaw,omitempty" json:"resultRaw,omitempty"`
	Exports             map[string]string `yaml:",omitempty" json:",omitempty"`
	// Result for IF node
	Condition            string `yaml:",omitempty" json:",omitempty"`
	ConditionEvaluatedTo bool   `yaml:"conditionEvaluatedTo,omitempty" json:"conditionEvaluatedTo,omitempty"`
	OnTrue               []N    `yaml:"onTrue,flow,omitempty" json:"onTrue,flow,omitempty"`
	OnFalse              []N    `yaml:"onFalse,omitempty,flow" json:"onFalse,omitempty,flow"`
	OnCondition          []N    `yaml:"onCondition,omitempty,flow" json:"onCondition,omitempty,flow"`
}

// Playbook represents a playbook tree that can be executed
type Playbook struct {
	FirstNode Node
}

// NewPlaybookFromYaml returns a new *Playbook from yaml string
func NewPlaybookFromYaml(yamlData []byte) (*Playbook, error) {
	var nodes []N = make([]N, 0, 10)
	err := yaml.Unmarshal(yamlData, &nodes)
	if err != nil {
		fmt.Printf("Error in unmarshalling: %s\n", err.Error())
		return nil, nil
	}
	// fmt.Printf("\n%v\n", nodes)

	return &Playbook{FirstNode: ConvertToLinkedNodes(nodes)}, nil
}

func UpdateWithResultStatus(nodes *[]N, state *ExecState) {
	if nodes == nil {
		return
	}

	ns := *nodes
	for i := 0; i < len(ns); i++ {
		// (*nodes)[i]
		nodeID := ns[i].ID
		switch ns[i].NodeType {
		case "execute":
			st, ok := state.actionResults[nodeID]
			if !ok {
				ns[i].State = "Not-Yet-Started"
				break
			}
			if st.done {
				ns[i].State = "Done"
				if st.actionResult.ResultJSON != "" {
					ns[i].RawResult = st.actionResult.ResultJSON
				} else {
					ns[i].ResultFields = st.actionResult.ResultFieldMap
				}
			}
			if st.waitingOnInput {
				ns[i].State = "Waiting-Var-Resolution"
			}

		case "if":
			stIf, ok := state.ifResults[nodeID]
			if !ok {
				ns[i].State = "Not-Yet-Started"
				break
			}
			if stIf.done {
				ns[i].State = "Done"
				ns[i].ConditionEvaluatedTo = stIf.evaluatedToTrue
			}
			if stIf.waitingOnInput {
				ns[i].State = "Waiting-Var-Resolution"
			}
			fmt.Printf("Type inside on True %T\n", ns[i].OnTrue)
			if stIf.evaluatedToTrue {
				UpdateWithResultStatus(&(ns[i].OnTrue), state)
			} else {
				UpdateWithResultStatus(&(ns[i].OnFalse), state)
			}

		}
	}

}

func ConvertToLinkedNodes(nodes []N) Node {
	if nodes == nil {
		return nil
	}

	var firstNode Node = nil
	var prevNode Node = nil
	for _, n := range nodes {
		if n.NodeType == "" {
			n.NodeType = "execute"
		}
		if n.ID == "" {
			n.ID = strconv.Itoa(count)
			count++
		}
		var currNode Node = nil
		switch n.NodeType {
		case "execute":
			currNode = &ActionNode{
				GenericExecutionNode: GenericExecutionNode{n.ID, ActionNodeT, nil},
				urn:                  n.Urn,
				inputParams:          n.Params,
				exportResultAs:       n.Exports,
			}

		case "if":
			currNode = &IfNode{
				GenericExecutionNode: GenericExecutionNode{n.ID, IfNodeT, nil},
				condition:            n.Condition,
				YesPathFirstNode:     ConvertToLinkedNodes(n.OnTrue),
				NoPathFirstNode:      ConvertToLinkedNodes(n.OnFalse),
			}

		case "for":
			// Under development.
		}

		if prevNode == nil {
			firstNode = currNode
		} else {
			prevNode.SetNext(currNode)
		}
		prevNode = currNode
	}

	return firstNode
}
