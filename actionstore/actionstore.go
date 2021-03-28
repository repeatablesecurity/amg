package actionstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
)

// ActionResult store the result of an Action execution
type ActionResult struct {
	ResultJSON     string            `json:"outputJson"`
	ResultFieldMap map[string]string `json:"outputFields"`
	ErrStr         string            `json:"error"`
}

type inputArgsToResultMapping struct {
	Input map[string]string `json:"input"`
	ActionResult
}

type actionMockScenario struct {
	ActionUrn             string                     `json:"actionUrn"`
	ExecutionDurationSecs int                        `json:"executionDuration"`
	Scenarios             []inputArgsToResultMapping `json:"scenarios"`
}

// ActionStore represents an instance of ActionStore containing a bunch of actions
type ActionStore struct {
	mockScenarios map[string]actionMockScenario
}

// NewActionStore creates a new instance of ActionStore
func NewActionStore(mockScenarioFile string) *ActionStore {
	jsonFile, err := os.Open(mockScenarioFile)
	if err != nil {
		fmt.Println("Error in opening mock scenarios file")
		return nil
	}
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		fmt.Println("Error in reading the mock scenario file")
		return nil
	}

	var s []actionMockScenario
	json.Unmarshal(byteValue, &s)

	//fmt.Printf("%+v", s)

	as := &ActionStore{mockScenarios: make(map[string]actionMockScenario)}
	for i := 0; i < len(s); i++ {
		urn := s[i].ActionUrn
		as.mockScenarios[urn] = s[i]
	}
	return as
}

// ExecuteAction exectes an action. It compares inputParams with mock scenarios
func (as *ActionStore) ExecuteAction(
	urn string, inputParams map[string]string) (*ActionResult, error) {
	fmt.Printf("Executing action with urn: %s\n", urn)
	// Check if a mock scenario exist for the urn
	if val, ok := as.mockScenarios[urn]; ok {
		for i := 0; i < len(val.Scenarios); i++ {
			scenario := &val.Scenarios[i]
			matched := true
			for ipKey, ipValue := range scenario.Input {
				if (ipValue != "*") && (ipValue != inputParams[ipKey]) {
					matched = false
					break
				}
			}
			if matched {
				// fmt.Print("Matched")
				copyAr := scenario.ActionResult
				return &copyAr, nil
			}
		}
	}

	return nil, errors.New("No scenarios found in mock data")
}
