package execution

import (
	"fmt"
	"strings"
	"time"

	"rptsec.com/amg/actionstore"
)

// ValueWrapper holds a value (as well as its type info) that is assigned to a variable.
type ValueWrapper struct {
	IsScalar bool
	// IsJson   bool
	S string
	V []string
}

// WrapStringValue wraps a string into a *ValueWrapper
func WrapStringValue(s string) *ValueWrapper {
	return &ValueWrapper{
		IsScalar: true,
		S:        s,
	}
}

// IterableString returns an array of string
func (v *ValueWrapper) IterableString() []string {
	if v.IsScalar {
		return []string{v.S}
	}
	return v.V
}

// IterableWrappedVal returns an array of *ValueWrapper
func (v *ValueWrapper) IterableWrappedVal() []*ValueWrapper {
	if v.IsScalar {
		return []*ValueWrapper{&ValueWrapper{IsScalar: true, S: v.S}}
	}
	r := make([]*ValueWrapper, len(v.V))
	for i := 0; i < len(v.V); i++ {
		r[i] = &ValueWrapper{IsScalar: true, S: v.V[i]}
	}
	return r
}

// StringVal gives the string value of the underlying object being held.
func (v *ValueWrapper) StringVal() string {
	if v.IsScalar {
		return v.S
	}
	return strings.Join(v.V, ",")
}

// AsString gives the string value of the underlying object being held.
func (v *ValueWrapper) AsString() string {
	return v.StringVal()
}

// -----------------------------------------------------------------------

// -----------------------------------------------------------------------

type ExecState struct {
	version        uint64
	execStartTs    int64
	execDoneTs     int64
	ErrStr         string
	lastUpdateTs   int64
	actionResults  map[string]*ActionExecState
	ifResults      map[string]*IfExecState
	forResults     map[string]*ForExecState
	exportedValues map[string]*ValueWrapper
	alertData      map[string]*ValueWrapper
}

// NewExecState creates new *ExecState
func NewExecState(initialVarValues map[string]string) *ExecState {
	initialBagOfVal := make(map[string]*ValueWrapper)
	for varName, val := range initialVarValues {
		initialBagOfVal[varName] = &ValueWrapper{IsScalar: true, S: val}
	}
	return &ExecState{
		version:        0,
		execStartTs:    time.Now().Unix(),
		lastUpdateTs:   time.Now().Unix(),
		actionResults:  make(map[string]*ActionExecState),
		ifResults:      make(map[string]*IfExecState),
		forResults:     make(map[string]*ForExecState),
		exportedValues: make(map[string]*ValueWrapper),
		alertData:      initialBagOfVal,
	}
}

// SetDoneWithError is called by a node to mark failure for that track.
func (e *ExecState) SetDoneWithError(errStr string) {
	e.ErrStr = errStr
	e.execDoneTs = time.Now().Unix()
	e.lastUpdateTs = e.execDoneTs
	e.version++
}

// GetVal returns a value associated with the variable name.
func (e *ExecState) GetVal(varName string) (*ValueWrapper, bool) {
	prefix := varName[0]
	switch prefix {
	case '@':
		// Look in alerts, and enrichment data
		if strings.HasPrefix(varName, "@alert:") {
			name := varName[len("@alert")+1:]
			fmt.Printf("Looking for var: %s in alert data %v\n", name, e.alertData)
			if val, ok := e.alertData[name]; ok {
				return val, true
			}
			return nil, false
		}
		if strings.HasPrefix(varName, "@node:") {
			nameWithNodeID := varName[len("@nodeId:")-2:]
			fmt.Printf("Looking for node value: %s\n", nameWithNodeID)
			nodeIDAndVarName := strings.Split(nameWithNodeID, "$")
			nodeID := nodeIDAndVarName[0]
			varName := nodeIDAndVarName[1]
			fmt.Printf("NodeID: |%s|, varName: |%s| in %v\n", nodeID, varName, e.actionResults)
			nodeState, ok := e.actionResults[nodeID]
			if !ok {
				return nil, false
			}
			fmt.Printf("Node state: %v", nodeState)
			if !nodeState.done {
				return nil, false
			}
			if nodeState.errStr != "" {
				return nil, false
			}
			if varName == "raw" {
				return &ValueWrapper{IsScalar: true, S: nodeState.actionResult.ResultJSON}, true
			}
			if val, ok := nodeState.actionResult.ResultFieldMap[varName]; ok {
				return &ValueWrapper{IsScalar: true, S: val}, true
			}
			return nil, false
		}

	case '$':
		// Look in exported vars
		varName = varName[1:]
		fmt.Printf("Looking for var: %s in %v\n", varName, e.exportedValues)
		if val, ok := e.exportedValues[varName]; ok {
			return val, true
		}
		return nil, false
	}

	// varName = varName[1:]
	// strings.Contains(varName, "##")
	// splitNames := strings.Split(varName, "##")
	// // Look in exported vars
	// if len(splitNames) == 1 {
	// 	exportedVarName := splitNames[0]
	// 	fmt.Printf("Looking for var: %s in %v\n", varName, e.exportedValues)
	// 	if val, ok := e.exportedValues[exportedVarName]; ok {
	// 		return val, true
	// 	} else {
	// 		return nil, false
	// 	}
	// }

	// // Look in node results
	// if len(splitNames) == 2 {
	// 	nodeID := splitNames[0]
	// 	resultField := splitNames[1]
	// 	ar := e.actionResults[nodeID]
	// 	if ar.errStr != "" {
	// 		return nil, false
	// 	}
	// 	if resultField == "raw" {
	// 		return &ValueWrapper{IsScalar: true, S: ar.actionResult.ResultJSON}, true
	// 	}
	// 	val, ok := ar.actionResult.ResultFieldMap[resultField]
	// 	if ok {
	// 		return &ValueWrapper{IsScalar: true, S: val}, true
	// 	}
	// 	fmt.Println("Request a field value thats not present after action successful completion")
	// 	return nil, false
	// }
	return nil, false
}

// NodeResult retrieves result for an executed node.
func (e *ExecState) NodeResult(nodeID string, fieldName string) (string, bool) {
	// fmt.Printf("Action result of the node: %+v", e.actionResults)
	valW, ok := e.actionResults[nodeID]
	if ok {
		return valW.actionResult.ResultFieldMap[fieldName], ok
	}
	return "", false
}

// -----------------------------------------------------------------------
// ******************** FOR-loop related methods *************************
//
// NOTE:
// Here is the list of calls that a For-Loop will make
//		startForLoop()
//						Called before first iteration starts.
//		startForLoopNextIteration()
//						Called before start of each iteration.
//						It also marks the end of previous iteration.
//		endForLoop()
//						Called after all iterations have ended.

// ForExecState holds data can be sent back to UI regarding execution of action node
type ForExecState struct {
	waitingOnInput              bool
	done                        bool
	currentIterationNumber      int
	currentIterationVal         *ValueWrapper
	lastExportedProgressVersion uint64
	currentIterationState       *ExecState
	iterationResultMap          map[int]*ExecState
	iterationValMap             map[int]*ValueWrapper
}

// Method to denote the start of execution for For Loop.
func (e *ExecState) startForLoop(fNode *ForNode, waitingOnInput bool) {
	e.forResults[fNode.Id] = &ForExecState{
		waitingOnInput:              waitingOnInput,
		done:                        false,
		currentIterationNumber:      -1,
		currentIterationVal:         nil,
		lastExportedProgressVersion: 0,
		currentIterationState:       nil,
	}
}

func (e *ExecState) startForLoopIteration(
	fNode *ForNode, itNum int, itVal *ValueWrapper, nestedState *ExecState) {
	var forState *ForExecState = e.forResults[fNode.Id]

	// Freeze and Store previous iteration state.
	if forState.currentIterationState != nil {
		forState.iterationResultMap[forState.currentIterationNumber] = forState.currentIterationState
		forState.iterationValMap[forState.currentIterationNumber] = forState.currentIterationVal
	}

	forState.waitingOnInput = false
	forState.currentIterationNumber = itNum
	forState.currentIterationVal = itVal
	forState.currentIterationState = nestedState

	// Increase the version so that the next export picks up the changes.
	forState.currentIterationState.version++
}

func (e *ExecState) endForLoop(nodeId string) {
	var forState *ForExecState = e.forResults[nodeId]
	forState.done = true
	// TODO: Export values into ExecState.
}

// ***************** End of FOR-loop related methods **************************
// ----------------------------------------------------------------------------

// ----------------------------------------------------------------------------
// ****************** ACTION Execution related methods ************************

// ActionExecState holds data can be sent back to UI regarding execution of action node
type ActionExecState struct {
	// Waiting on input arg's values to be available
	waitingOnInput bool
	// Set when execution is finished
	done bool
	// Error string
	errStr string
	// Resolved values for parameters needed to execute the action
	concreteInputParams map[string]string
	// Whats this for?
	usedValues map[string]string
	// Result of the action execution
	actionResult *actionstore.ActionResult
}

func (e *ExecState) startActionExecution(id string) {
	actionES := &ActionExecState{
		waitingOnInput:      true,
		done:                false,
		concreteInputParams: nil,
		actionResult:        nil,
	}
	e.actionResults[id] = actionES
	// fmt.Printf("Starting execution for action: %+s\n", id)
}

// UpdateConcreteParamsForActionExecution is called when the action has resolved an input param.
// It can be called each time an input param is resolved, or called once when all have been resolved.
func (e *ExecState) UpdateConcreteParamsForActionExecution(
	id string, valMap map[string]string, allParamsResolved bool) {
	actionES := e.actionResults[id]
	actionES.waitingOnInput = !allParamsResolved
	if actionES.concreteInputParams == nil {
		actionES.concreteInputParams = make(map[string]string)
	}
	for varName, resolvedVal := range valMap {
		actionES.concreteInputParams[varName] = resolvedVal
	}
}

func (e *ExecState) updateSuccessResultForAction(
	id string, result *actionstore.ActionResult, exports map[string]*ValueWrapper) {
	// fmt.Println("Updating success result for action")
	actionES := e.actionResults[id]
	actionES.actionResult = result
	actionES.done = true

	// Use the 'exports' to update the exports at the execState level
	for k, v := range exports {
		e.exportedValues[k] = v
	}
}

func (e *ExecState) updateErrorResultForAction(id string, errStr string) {
	actionES := e.actionResults[id]
	actionES.errStr = errStr
}

// ************** End of ACTION execution related methods **********************
// -----------------------------------------------------------------------------

// IfExecState holds data can be sent back to UI regarding execution of action node
type IfExecState struct {
	waitingOnInput    bool
	done              bool
	errStr            string
	concreteCondition string
	varValues         map[string]string
	evaluatedToTrue   bool
}

func (e *ExecState) startIfNodeExecution(id string) {
	e.ifResults[id] = &IfExecState{
		true, false, "", "", make(map[string]string), false}
	e.version++
}

func (e *ExecState) updateIfNodeEvaluation(id string, yesPath bool, varValuesUsed map[string]string) {
	ifState := e.ifResults[id]
	ifState.done = true
	ifState.varValues = varValuesUsed
	ifState.evaluatedToTrue = yesPath

	e.version++
}

func (e *ExecState) updateIfNodeEvaluationError(id string, errStr string) {
	ifState := e.ifResults[id]
	ifState.done = true
	ifState.errStr = errStr

	e.version++
}
