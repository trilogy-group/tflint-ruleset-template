package main

import (
	"bufio"
	"os"
	"os/exec"
	"strings"

	"github.com/terraform-linters/tflint-plugin-sdk/plugin"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
	"github.com/terraform-linters/tflint-ruleset-template/rules"
)

func readReccosFile(fileName string) (map[string]map[string]string, error) {
	reccosMap := map[string]map[string]string{}
	file, err := os.Open(fileName)
	if err != nil {
		return reccosMap, err //System fail
	}

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		items := strings.Split(line, ":") //items[0] -> AWSID, items[1] -> attributeType items[2] -> attributeValue
		innerMap, exists := reccosMap[items[0]]
		if !exists {
			tempMap := make(map[string]string) // a new map will have to be made as a map with the given AWSID does not exist
			tempMap[items[1]] = items[2]
			reccosMap[items[0]] = tempMap
		} else {
			innerMap[items[1]] = items[2]
			reccosMap[items[0]] = innerMap
		}
	}
	return reccosMap, nil
}

func readTagFile(fileName string) (map[string]string, error) {
	tagMap := map[string]string{}
	file, err := os.Open(fileName)
	if err != nil {
		return tagMap, err //System fail
	}

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		items := strings.Split(line, ":") //items[0] -> tag items[1] -> AWSID
		tagMap[items[0]] = items[1]
	}
	return tagMap, nil
}

func main() {
	reccosFileName := os.Getenv("ReccosMapFile")
	tagFileName := os.Getenv("TagsMapFile") //both of these environment variables would have been set by the orchestrator
	currPWD, err := exec.Command("pwd").Output()
	if err != nil {
		//Add error log
		panic(err) //system crash
	}
	currPWDStrip := strings.Trim(string(currPWD), "\n") //there is a new line char by defualt that needs to be trimmed
	reccosFilePath := currPWDStrip + "/" + reccosFileName
	reccos, errR := readReccosFile(reccosFilePath)
	if errR != nil {
		//Add error log
		panic(errR) //system fail
	}
	tagFilePath := currPWDStrip + "/" + tagFileName
	tagToID, errT := readTagFile(tagFilePath)
	if errT != nil {
		//Add err log
		panic(errT) //system fail
	}
	var taggableMap = make(map[string]bool)
	for _, resourceType := range taggableArray {
		taggableMap[resourceType] = true
	}
	plugin.Serve(&plugin.ServeOpts{
		RuleSet: &tflint.BuiltinRuleSet{
			Name:    "template",
			Version: "0.1.0",
			Rules: []tflint.Rule{
				rules.NewReccomendationFlagRule(tagToID, reccos, taggableMap),
				rules.NewGetModuleSourceRule(),
			},
		},
	})
}
