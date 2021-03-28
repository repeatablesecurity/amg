package execution

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"rptsec.com/amg/actionstore"
)

func TestExecState_Basic(t *testing.T) {
	assert := assert.New(t)

	exSt := NewExecState(nil)
	assert.NotNil(exSt)
	assert.Equal(exSt.version, uint64(0))

	exSt.startActionExecution("action-node-1")
	ar := &actionstore.ActionResult{ResultFieldMap: map[string]string{"reputationScore": "50"}}
	exSt.updateSuccessResultForAction("action-node-1", ar, nil)

	val, ok := exSt.NodeResult("action-node-1", "reputationScore")
	assert.True(ok)
	assert.Equal(val, "50")

}
