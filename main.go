package main

import (
	"github.com/terraform-linters/tflint-plugin-sdk/plugin"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
	"github.com/terraform-linters/tflint-ruleset-template/rules"
)

func main() {
	tagToID := make(map[string]string)
	reccos := make(map[string]map[string]string)
	plugin.Serve(&plugin.ServeOpts{
		RuleSet: &tflint.BuiltinRuleSet{
			Name:    "template",
			Version: "0.1.0",
			Rules: []tflint.Rule{
				rules.NewReccomendationFlagRule(tagToID, reccos),
				rules.NewGetModuleSourceRule(),
			},
		},
	})
}
