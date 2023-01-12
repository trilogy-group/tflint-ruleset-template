package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/terraform-linters/tflint-plugin-sdk/plugin"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
	"github.com/terraform-linters/tflint-ruleset-template/rules"
	"github.com/trilogy-group/cloudfix-linter/cloudfixIntegration"
)

func readReccosFile(fileName string) (map[string]cloudfixIntegration.Recommendation, error) {
	reccosMap := map[string]cloudfixIntegration.Recommendation{}
	data, err := os.ReadFile(fileName)
	if err != nil {
		return reccosMap, err
	}
	err = json.Unmarshal(data, &reccosMap)
	if err != nil {
		return reccosMap, err
	}
	return reccosMap, nil
}


func readTagFile(fileName string) (map[string]map[string][]string, error) {
	tagMap := map[string]map[string][]string{}
	data, err := os.ReadFile(fileName)
	if err != nil {
		return tagMap, err
	}
	err = json.Unmarshal(data, &tagMap)
	if err != nil {
		return tagMap, err
	}
	return tagMap, nil
}

func main() {
	reccosFileName := os.Getenv("ReccosMapFile")
	tagFileName := os.Getenv("TagsMapFile") //both of these environment variables would have been set by the orchestrator
	currPWDStrip := ""
	reccosFilePath := ""
	tagFilePath := ""
	if runtime.GOOS == "windows" {
		currPWD, err := exec.Command("powershell", "-NoProfile", "(pwd).path").Output()
		if err != nil {
			panic(err)
		}
		currPWDStrip = strings.Trim(string(currPWD), "\n")
		currPWDStrip = strings.TrimSuffix(currPWDStrip, "\r")
		reccosFilePath = currPWDStrip + "\\" + reccosFileName
		tagFilePath = currPWDStrip + "\\" + tagFileName
	} else {
		currPWD, err := exec.Command("pwd").Output()
		if err != nil {
			panic(err)
		}
		currPWDStrip = strings.Trim(string(currPWD), "\n")
		reccosFilePath = currPWDStrip + "/" + reccosFileName
		tagFilePath = currPWDStrip + "/" + tagFileName
	}
	reccos, errR := readReccosFile(reccosFilePath)
	if errR != nil {
		panic(errR)
	}

	tagToID, errT := readTagFile(tagFilePath)
	if errT != nil {
		panic(errT)
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
