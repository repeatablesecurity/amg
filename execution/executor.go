package execution

import (
	"fmt"
	"strings"
	"time"

	"rptsec.com/amg/actionstore"
)

// Execution represents an instance of playbook execution
type Execution struct {
	startNode             Node
	as                    *actionstore.ActionStore
	execState             *ExecState
	startTs               int64
	doneTs                int64
	err                   bool
	errStr                string
	totalActionExecutions int
}

// NewExecution creates an instance of Execution
func NewExecution(p *Playbook, initialVarValues map[string]string, mockScenarioFile string) *Execution {
	as := actionstore.NewActionStore(mockScenarioFile)
	if as == nil {
		_, _ = fmt.Println("error in creating ActionStore instance")
		return nil
	}
	return &Execution{p.FirstNode, as, NewExecState(initialVarValues), 0, 0, false, "", 0}
}

// Start the execution
func (ex *Execution) Start() {
	// Print the urns of the action nodes.
	ex.startTs = time.Now().Unix()
	stack := []*ExecState{ex.execState}
	execStateStack := NewExecStateStack(stack)
	ex.executeSeriallyFrom(ex.startNode, execStateStack)
}

// executeSeriallyFrom executes Node n and all nodes that follow it serially
// It will block if a node needs to wait for a trigger or input
func (ex *Execution) executeSeriallyFrom(n Node, execStateStack *ExecStateStack) bool {
	for currNode := n; currNode != nil; currNode = currNode.Next() {
		switch currNode.NodeType() {
		case ActionNodeT:
			ex.totalActionExecutions++
			var ac *ActionNode
			ac = currNode.(*ActionNode)
			fmt.Printf("Count# %d Action node urn: %s with params\n +%+v \n",
				ex.totalActionExecutions, ac.urn, ac.inputParams)
			if ex.executeAction(ac, execStateStack) == false {
				return false
			}

		case IfNodeT:
			var ifNode = currNode.(*IfNode)
			if ex.executeIfBlock(ifNode, execStateStack) == false {
				return false
			}

		case ForNodeT:
			var forNode = currNode.(*ForNode)
			if ex.executeForLoop(forNode, execStateStack) == false {
				return false
			}
		}
	}
	return true // Reaching here means that n was passed a nil
}

func (ex *Execution) executeIfBlock(ifNode *IfNode, execStateStack *ExecStateStack) bool {
	execState := execStateStack.Top()
	execState.startIfNodeExecution(ifNode.Id)
	c := NewCondition(ifNode.condition)
	// Resolve the list of variables needed to evaluating the if condition
	varValuesUsed := make(map[string]string)
	for _, varName := range c.UnknownVarsList() {
		valW, _ := execStateStack.GetValueOrBlock(varName, 100)
		c.SetVarValue(varName, valW)
		varValuesUsed[varName] = valW.AsString()
	}
	// Evaluate the condition
	yesPath, success := c.Evaluate()
	// Update the if node state
	if !success {
		execState.updateIfNodeEvaluationError(ifNode.Id, "Evaluation failed")
		return false
	}
	execState.updateIfNodeEvaluation(ifNode.Id, yesPath, varValuesUsed)
	// Continue to execute the yesPath or noPath
	var ret bool = true
	if yesPath {
		var yNode Node = ifNode.YesPathFirstNode
		ret = ex.executeSeriallyFrom(yNode, execStateStack)
	} else {
		if ifNode.NoPathFirstNode != nil {
			ret = ex.executeSeriallyFrom(ifNode.NoPathFirstNode, execStateStack)
		}
	}
	return ret
}

func (ex *Execution) executeAction(n *ActionNode, execStateStack *ExecStateStack) bool {
	topExecState := execStateStack.Top()
	topExecState.startActionExecution(n.Id)

	var inputParamsConcrete map[string]string = make(map[string]string)
	for paramName, val := range n.inputParams {
		if !strings.HasPrefix(val, "@") && !strings.HasPrefix(val, "$") {
			inputParamsConcrete[paramName] = val
			continue
		}
		concreteVal, present := execStateStack.GetValueOrBlock(val, 1000)
		if !present {
			topExecState.updateErrorResultForAction(n.Id, "Timed out waiting for dependency")
			return false
		}
		inputParamsConcrete[paramName] = concreteVal.StringVal()
	}

	topExecState.UpdateConcreteParamsForActionExecution(n.Id, inputParamsConcrete, true)
	ar, err := ex.as.ExecuteAction(n.urn, inputParamsConcrete)
	if err != nil {
		topExecState.updateErrorResultForAction(n.Id, err.Error())
		return false
	}

	// fmt.Printf("Action result after execution %+v", ar)
	// Compute the exportAs values.
	exports := ex.evaluateExportsForAction(n.exportResultAs, ar)
	execStateStack.ExportUp(exports)

	topExecState.updateSuccessResultForAction(n.Id, ar, exports)
	// TODO: Above, exports is added twice to the top stack. Fix it.
	return true
}

func (ex *Execution) evaluateExportsForAction(
	toExport map[string]string, ar *actionstore.ActionResult) map[string]*ValueWrapper {
	var exports map[string]*ValueWrapper = make(map[string]*ValueWrapper)
	for exportedName, expr := range toExport {
		if expr == "result" {
			exports[exportedName] = WrapStringValue(ar.ResultJSON)
		} else {
			if val, ok := ar.ResultFieldMap[expr]; ok {
				exports[exportedName] = WrapStringValue(val)
			}
		}
	}
	return exports
}

// DONE
func (ex *Execution) executeForLoop(n *ForNode, executeStateStack *ExecStateStack) bool {
	// ExecState that is on top of the stack
	topState := executeStateStack.Top()
	topState.startForLoop(n, true /* waiting on input */)

	// Get the var on which the loop will iterate
	iterableVar := n.IterateOnVar
	// Block on the var to be available
	val, present := executeStateStack.GetValueOrBlock(iterableVar, 120 /* timeoutSecs */)
	// Error on timeout if var is not available
	if !present {
		topState.SetDoneWithError("Timed out waiting on input dependencies")
		return false
	}

	iterableVal := val.IterableWrappedVal()
	loopCount := len(iterableVal)
	var firstNode Node = n.FirstLoopNode
	for i := 1; i <= loopCount; i++ {
		val := iterableVal[i]
		nestedState := NewExecState(nil)
		topState.startForLoopIteration(n, i, val, nestedState)

		newStackForLoop := executeStateStack.NewNestedStack()
		ex.executeSeriallyFrom(firstNode, newStackForLoop)
		// TODO: check for error from executeSeriallyFrom()
	}

	topState.endForLoop(n.Id)
	return true
}

func (ex *Execution) Status(lastConsumedVersion int64) *ExecState {
	copy := *ex.execState
	return &copy
}
