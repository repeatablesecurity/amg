package execution

import "time"

type ExecStateStack struct {
	execStates []*ExecState
}

// NewExecStateStack maintains a stack of ExecState and provides read/write
// functionality over them based on visibility configuration
// Current implementation uses defalt config such that -
// - Reads will search for an exported var in the bottom most ExecState and look upwards
// - Writes will limited to the bottom most ExecState
func NewExecStateStack(execStates []*ExecState) *ExecStateStack {
	return &ExecStateStack{execStates: execStates}
}

// NewNestedStack adds a new *ExecState and returns a new *ExecStateStack
func (st *ExecStateStack) NewNestedStack() *ExecStateStack {
	newStates := st.execStates
	newStates[len(newStates)+1] = NewExecState(nil)
	return &ExecStateStack{execStates: newStates}
}

// GetValue gets a value if present, else returns immediately
func (st *ExecStateStack) GetValue(name string) (*ValueWrapper, bool) {
	for i := len(st.execStates) - 1; i >= 0; i-- {
		execState := st.execStates[i]
		val, present := execState.GetVal(name)
		if present {
			return val, true
		}
	}
	return nil, false
}

// GetValueOrBlock blocks the caller until value is available
func (st *ExecStateStack) GetValueOrBlock(name string, timeoutSecs int) (*ValueWrapper, bool) {
	for totalWaitSeconds := 0; totalWaitSeconds <= timeoutSecs; totalWaitSeconds += 2 {
		if val, present := st.GetValue(name); present {
			return val, present
		}
		time.Sleep(2 * time.Millisecond)
	}
	return nil, false
}

// ExportUp exports a var-value pair to upper levels
func (st *ExecStateStack) ExportUp(exports map[string]*ValueWrapper) {
	for _, execSt := range st.execStates {
		for varName, varValue := range exports {
			execSt.exportedValues[varName] = varValue
		}
	}
}

// Top returns the topmost *ExecState in the stack
func (st *ExecStateStack) Top() *ExecState {
	return st.execStates[len(st.execStates)-1]
}
