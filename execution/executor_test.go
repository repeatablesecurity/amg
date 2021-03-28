package execution

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasic(t *testing.T) {
	assert := assert.New(t)
	// Create two action nodes, and link them.

	alertData := map[string]string{"srcIp": "192.168.0.1", "dstIp": "192.168.0.2"}
	var ac1 *ActionNode
	ac1 = &ActionNode{
		GenericExecutionNode: GenericExecutionNode{"ac1", ActionNodeT, nil},
		urn:                  "www.vt.com/soar-services/v1/checkIpReputation",
		inputParams:          map[string]string{"ipv4Addr": "@alert:srcIp"},
		exportResultAs:       map[string]string{"n1_reputationScore": "reputationScore"},
	}

	var ac2 *ActionNode
	ac2 = &ActionNode{
		GenericExecutionNode: GenericExecutionNode{"ac2", ActionNodeT, nil},
		urn:                  "www.vt.com/soar-services/v1/checkIpReputation",
		inputParams:          map[string]string{"ipv4Addr": "@alert:dstIp"},
		exportResultAs:       map[string]string{"n2_reputationScore": "reputationScore"},
	}
	ac1.SetNext(ac2)

	var if3 *IfNode = &IfNode{
		GenericExecutionNode: GenericExecutionNode{"if3", IfNodeT, nil},
		condition:            "@node:ac1$reputationScore == 50",
		YesPathFirstNode:     nil,
		NoPathFirstNode:      nil,
	}
	ac2.SetNext(if3)

	var ifYesAc1 *ActionNode = &ActionNode{
		GenericExecutionNode: GenericExecutionNode{"ifYesAc1", ActionNodeT, nil},
		urn:                  "www.rptsec.com/sms/v1/getDomainForIp",
		inputParams:          map[string]string{"ipv4Addr": "@alert:srcIp"},
	}
	if3.YesPathFirstNode = ifYesAc1
	var ifYesAc2 *ActionNode = &ActionNode{
		GenericExecutionNode: GenericExecutionNode{"ifYesAc2", ActionNodeT, nil},
		urn:                  "www.vt.com/soar-services/v1/checkDomainReputation",
		inputParams:          map[string]string{"domainName": "@node:ifYesAc1$domainName"},
	}
	ifYesAc1.SetNext(ifYesAc2)

	var ifNoAc1 *ActionNode = &ActionNode{
		GenericExecutionNode: GenericExecutionNode{"ifNoAc1", ActionNodeT, nil},
		urn:                  "www.vt.com/soar-services/v1/checkIpReputation",
		inputParams:          map[string]string{"ipv4Addr": "192.168.0.4"},
		exportResultAs:       map[string]string{"n4_reputationScore": "reputationScore"},
	}
	if3.NoPathFirstNode = ifNoAc1

	var ac4 *ActionNode
	ac4 = &ActionNode{
		GenericExecutionNode: GenericExecutionNode{"ac4", ActionNodeT, nil},
		urn:                  "www.vt.com/soar-services/v1/checkIpReputation",
		inputParams:          map[string]string{"ipv4Addr": "192.168.0.5"},
		exportResultAs:       map[string]string{"n5_reputationScore": "reputationScore"},
	}
	if3.SetNext(ac4)

	ex := NewExecution(&Playbook{ac1}, alertData, "../actionstore/action-input-output.json")
	assert.NotNil(ex)
	ex.Start()
	assert.NotNil(ex.execState)
	execState := ex.execState
	ac1Score, ok := execState.NodeResult("ac1", "reputationScore")
	assert.True(ok)
	assert.Equal(ac1Score, "50")

	ac2Score, ok := execState.NodeResult("ac2", "reputationScore")
	assert.True(ok)
	assert.Equal(ac2Score, "52")

	ifYesAc2Score, ok := execState.NodeResult("ifYesAc2", "reputationScore")
	assert.True(ok)
	assert.Equal(ifYesAc2Score, "90")
}
