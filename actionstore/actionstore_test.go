package actionstore

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasic(t *testing.T) {
	assert := assert.New(t)
	as := NewActionStore("./action-input-output.json")
	assert.NotNil(as)

	result, err := as.ExecuteAction(
		"www.vt.com/soar-services/v1/checkIpReputation",
		map[string]string{
			"ipv4Addr": "192.168.0.1",
		})
	assert.Nil(err)
	assert.Equal(result.ResultFieldMap["reputationScore"], "50")
}
