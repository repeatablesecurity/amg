package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"

	y "github.com/ghodss/yaml"
	// TODO: Use this for yaml parsing. Its a fork of below - "sigs.k8s.io/yaml"
	"gopkg.in/yaml.v2" // https://github.com/go-yaml/yaml

	"rptsec.com/amg/execution"
)

func main() {
	fmt.Println("My favorite number is", rand.Intn(10))
	mockScenariosFile := flag.String("mock-scenario-file", "resources/sample-mock-scenario.json", "File with scenarios that Action-Store should mock")
	playbookFile := flag.String("playbook", "resources/sample-playbook.yaml", "Playbook file that will be executed")
	alertsDataFile := flag.String("alert-data-file", "resources/sample-alert-data.json", "File that contains the alert data")
	resultFile := flag.String("result-file", "/tmp/result.yaml", "File where the result will be written")

	yamlData, err := ioutil.ReadFile(*playbookFile)
	if err != nil {
		fmt.Printf("Error reading %s, Err: %s\n", *playbookFile, err.Error())
		return
	}
	var yamlNodes []execution.N = make([]execution.N, 0, 10)
	err = yaml.Unmarshal(yamlData, &yamlNodes)
	if err != nil {
		fmt.Printf("Error in unmarshalling: %s\n", err.Error())
		return
	}

	e, _ := yaml.Marshal(yamlNodes)
	fmt.Printf("Nodes into Yaml as-is: \n%s", string(e))

	var playbook *execution.Playbook = &execution.Playbook{
		FirstNode: execution.ConvertToLinkedNodes(yamlNodes)}
	if playbook == nil {
		fmt.Printf("Error in building playbook from yaml nodes\n")
		return
	}

	alertsData, err := ioutil.ReadFile(*alertsDataFile)
	if err != nil {
		fmt.Printf("Error reading %s, Err: %s\n", *alertsDataFile, err.Error())
		return
	}
	var initialValues map[string]string
	if err := json.Unmarshal(alertsData, &initialValues); err != nil {
		fmt.Printf("Unable to parse the alerts data, Err: %s\n", err.Error())
		return
	}

	fmt.Printf("Executing playbook at: %s, for alert data at: %s, with mock scenarios at: %s",
		*playbookFile, *alertsDataFile, *mockScenariosFile)
	ex := execution.NewExecution(playbook, initialValues, *mockScenariosFile)
	ex.Start()
	es := ex.Status(1)
	execution.UpdateWithResultStatus(&yamlNodes, es)

	d, err := y.Marshal(yamlNodes)
	if err != nil {
		fmt.Printf("Its fatal: %v", err)
	}
	fmt.Printf("--- t dump:\n%s\n\n", string(d))

	j, err := json.Marshal(yamlNodes)
	if err == nil {
		fmt.Printf("\n\n JSON: \n %s", string(j))
	}

	ioutil.WriteFile(*resultFile, d, os.ModeAppend)
	fmt.Printf("\nDONE\n")
}
