package rules

import (
	"fmt"
	"strings"

	"github.com/terraform-linters/tflint-plugin-sdk/hclext"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
)

// AwsS3BucketExampleLifecycleRule checks whether ...
type ReccomendationFlagRule struct {
	tflint.DefaultRule
	TagToID        map[string]string
	AttributeRecco map[string]map[string]string
}

// NewAwsS3BucketExampleLifecycleRule returns a new rule
func NewReccomendationFlagRule(tagIDMap map[string]string, reccoMap map[string]map[string]string) *ReccomendationFlagRule {
	return &ReccomendationFlagRule{
		TagToID:        tagIDMap,
		AttributeRecco: reccoMap,
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

// Check checks whether ...
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
			continue
		}
		var getTags map[string]string
		_ = runner.EvaluateExpr(tags.Expr, &getTags, nil)

		var yor_trace = getTags["yor_trace"]
		yorTraceStrip := strings.Trim(yor_trace, "\n")
		yorTraceTrim := strings.Trim(yorTraceStrip, `"`)
		var AWSID = r.TagToID[yorTraceTrim]
		err = runner.EnsureNoError(err, func() error {
			if AWSID == "" {
				runner.EmitIssue(
					r,
					fmt.Sprintf("Failed to find AWS ID with yor_trace: \"%s\".Either the resource has not been deployed, or the yor trace has been changed. You might want to run terraform apply", yorTraceTrim),
					tags.Expr.Range(),
				)
			}
			return nil
		})
		if err != nil {
			return err
		}
		reccoforID := r.AttributeRecco[AWSID]
		for attributeType, attributeValue := range reccoforID {
			if attributeType == "NoAttributeMarker" {
				runner.EmitIssue(
					r,
					fmt.Sprintf("Oppurtunity Description: \"%s\"", attributeValue),
					module.DefRange,
				)
			} else {
				attributeTerraform, existsAttribute := module.Body.Attributes[attributeType]
				if !existsAttribute {
					runner.EmitIssue(
						r,
						fmt.Sprint("Oppurtunity exists but attribute not found. \"%s\" should be \"%s\"", attributeType, attributeValue),
						module.DefRange,
					)
				}
				var extractAttribute string
				runner.EvaluateExpr(attributeTerraform.Expr, &extractAttribute, nil)
				if extractAttribute != attributeValue {
					runner.EmitIssue(
						r,
						fmt.Sprintf("Reccomendation exists for this attribute. It should be set to \"%s\"", attributeValue),
						attributeTerraform.Expr.Range(),
					)
				}
			}
		}
	}
	return nil
}
