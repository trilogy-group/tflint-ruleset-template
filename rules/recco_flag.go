package rules

import (
	"fmt"
	"strings"

	"github.com/terraform-linters/tflint-plugin-sdk/hclext"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
)

// ReccomendationFlagRule flags of cloudifx reccommendations
type ReccomendationFlagRule struct {
	tflint.DefaultRule
	TagToID        map[string]string
	AttributeRecco map[string]map[string]string
	Taggable       map[string]bool
}

// Constructor for maaking the rule struct
func NewReccomendationFlagRule(tagIDMap map[string]string, reccoMap map[string]map[string]string, taggableMap map[string]bool) *ReccomendationFlagRule {
	return &ReccomendationFlagRule{
		TagToID:        tagIDMap,
		AttributeRecco: reccoMap,
		Taggable:       taggableMap,
	}
}

// Name returns the rule name
func (r *ReccomendationFlagRule) Name() string {
	return "flag_reccomend"
}

// Enabled returns whether the rule is enabled by default
func (r *ReccomendationFlagRule) Enabled() bool {
	return true
}

// Severity returns the rule severity
func (r *ReccomendationFlagRule) Severity() tflint.Severity {
	return tflint.WARNING
}

// Link returns the rule reference link
func (r *ReccomendationFlagRule) Link() string {
	return ""
}

//gives the list of attributes that the runner needs to extract
func (r *ReccomendationFlagRule) getAttributeList() []string {
	var attributes []string
	for _, reccos := range r.AttributeRecco {
		for attribute := range reccos {
			if attribute == "NoAttributeMarker" {
				continue
			}
			attributes = append(attributes, attribute)
		}
	}
	attributes = append(attributes, "tags")
	return attributes
}

// Check flags off cloudfix recommendations.
func (r *ReccomendationFlagRule) Check(runner tflint.Runner) error {
	var attributes = r.getAttributeList()
	var schema []hclext.AttributeSchema
	for _, attribute := range attributes {
		var temp hclext.AttributeSchema
		temp.Name = attribute
		schema = append(schema, temp)
	}
	resources, err := runner.GetModuleContent(&hclext.BodySchema{
		Blocks: []hclext.BlockSchema{
			{
				Type:       "resource",
				LabelNames: []string{"resource_type", "resource_name"},
				Body: &hclext.BodySchema{
					Attributes: schema,
				},
			},
		},
	}, nil)
	if err != nil {
		return err
	}
	for _, module := range resources.Blocks {
		tags, exists := module.Body.Attributes["tags"]
		if !exists {
			_, ok := r.Taggable[module.Labels[0]]
			if ok {
				runner.EmitIssue(
					r,
					"This resources is missing tags. Fix by running \"cloudfix-linter addTags\" followed by \"terraform apply\"!",
					module.DefRange,
				)
			}
			continue
		}
		var getTags map[string]string
		_ = runner.EvaluateExpr(tags.Expr, &getTags, nil)

		var yor_trace, foundY = getTags["yor_trace"]
		if !foundY {
			runner.EmitIssue(
				r,
				"This resource is missing a trace id tag. Fix by running \"cloudfix-linter addTags\" followed by \"terraform apply\"!",
				tags.Expr.Range(),
			)
			continue
		}
		yorTraceStrip := strings.Trim(yor_trace, "\n")
		yorTraceTrim := strings.Trim(yorTraceStrip, `"`)
		var AWSID, foundA = r.TagToID[yorTraceTrim]
		if !foundA {
			runner.EmitIssue(
				r,
				fmt.Sprintf("Couldn't find a matching AWS resource: \"%s\".Either it hasn't been deployed, or the trace ID has been changed. Run \"terraform apply\"!", yorTraceTrim),
				tags.Expr.Range(),
			)
			continue
		}
		AWS_Strip := strings.Trim(AWSID, "\n")
		AWSTrim := strings.Trim(AWS_Strip, `"`)
		reccoforID := r.AttributeRecco[AWSTrim]
		for attributeType, attributeValue := range reccoforID {
			if attributeType == "NoAttributeMarker" {
				runner.EmitIssue(
					r,
					fmt.Sprintf("Description: \"%s\"", attributeValue),
					module.DefRange,
				)
			} else {
				attributeTerraform, existsAttribute := module.Body.Attributes[attributeType]
				if !existsAttribute {
					runner.EmitIssue(
						r,
						fmt.Sprintf("Reduce cost by setting the value of attribute \"%s\" to \"%s\"", attributeType, attributeValue),
						module.DefRange,
					)
					continue
				}
				var extractAttribute string
				runner.EvaluateExpr(attributeTerraform.Expr, &extractAttribute, nil)
				if extractAttribute != attributeValue {
					runner.EmitIssue(
						r,
						fmt.Sprintf("Reduce cost by setting this value to \"%s\"", attributeValue),
						attributeTerraform.Expr.Range(),
					)
				}
			}
		}
	}
	return nil
}
