package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/terraform-linters/tflint-plugin-sdk/plugin"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
	"github.com/terraform-linters/tflint-ruleset-template/rules"
)

func readReccosFile(fileName string) (map[string]map[string][]string, error) {
	reccosMap := map[string]map[string][]string{}
	file, err := os.Open(fileName)
	if err != nil {
		return reccosMap, err
	}

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		items := strings.Split(line, "->") //items[0] -> AWSID, items[1] -> attributeType items[2] -> attributeValue
		if len(items) < 2 {
			return reccosMap, errors.New("Corrupt Recommendation")
		}
		innerMap, exists := reccosMap[items[0]]
		if !exists {
			tempMap := make(map[string][]string) // a new map will have to be made as a map with the given AWSID does not exist
			for i := 2; i < len(items); i++ {
				tempMap[items[1]] = append(tempMap[items[1]], items[i])
			}
			reccosMap[items[0]] = tempMap
		} else {
			for i := 2; i < len(items); i++ {
				innerMap[items[1]] = append(innerMap[items[1]], items[i])
			}
			reccosMap[items[0]] = innerMap
		}
	}
	return reccosMap, nil
}

func readTagFile(fileName string) (map[string]map[string]string, error) {
	tagMap := map[string]map[string]string{}
	file, err := os.Open(fileName)
	if err != nil {
		return tagMap, err //System fail
	}

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		items := strings.Split(line, "->") //items[0] -> tag items[1] -> AWSID
		if len(items) < 3 {
			return tagMap, errors.New("Invalid yor_tag in resource")
		}
		_, exists := tagMap[items[0]]
		if !exists {
			tagMap[items[0]] = map[string]string{}
		}
		tagMap[items[0]][items[1]] = items[2]
	}
	return tagMap, nil
}

func main() {
	reccosFileName := os.Getenv("ReccosMapFile")
	tagFileName := os.Getenv("TagsMapFile")
	currPWDStrip := ""
	reccosFilePath := ""
	tagFilePath := ""
	if runtime.GOOS == "windows" {
		currPWD, err := exec.Command("powershell", "-NoProfile", "(pwd).path").Output()
		if err != nil {
			fmt.Println(err)
			return
		}
		currPWDStrip = strings.Trim(string(currPWD), "\n")
		currPWDStrip = strings.TrimSuffix(currPWDStrip, "\r")
		reccosFilePath = currPWDStrip + "\\" + reccosFileName
		tagFilePath = currPWDStrip + "\\" + tagFileName
	} else {
		currPWD, err := exec.Command("pwd").Output()
		if err != nil {
			fmt.Println(err)
			return
		}
		currPWDStrip = strings.Trim(string(currPWD), "\n")
		reccosFilePath = currPWDStrip + "/" + reccosFileName
		tagFilePath = currPWDStrip + "/" + tagFileName
	}
	reccos, errR := readReccosFile(reccosFilePath)
	if errR != nil {
		fmt.Println(errR)
		return
	}

	tagToID, errT := readTagFile(tagFilePath)
	if errT != nil {
		fmt.Println(errT)
		return
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
