package rules

import (
	"fmt"

	"github.com/terraform-linters/tflint-plugin-sdk/hclext"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
)

// TerraformBackendTypeRule checks whether ...
type GetModuleSourceRule struct {
	tflint.DefaultRule
}

// NewTerraformBackendTypeRule returns a new rule
func NewGetModuleSourceRule() *GetModuleSourceRule {
	return &GetModuleSourceRule{}
}

// Name returns the rule name
func (r *GetModuleSourceRule) Name() string {
	return "module_source"
}

// Enabled returns whether the rule is enabled by default
func (r *GetModuleSourceRule) Enabled() bool {
	return false
}

// Severity returns the rule severity
func (r *GetModuleSourceRule) Severity() tflint.Severity {
	return tflint.NOTICE
}

// Link returns the rule reference link
func (r *GetModuleSourceRule) Link() string {
	return ""
}

// Check checks whether ...
func (r *GetModuleSourceRule) Check(runner tflint.Runner) error {
	// This rule is an example to get attributes of blocks other than resources.
	content, err := runner.GetModuleContent(&hclext.BodySchema{
		Blocks: []hclext.BlockSchema{
			{
				Type:       "module",
				LabelNames: []string{"terraform_name"},
				Body: &hclext.BodySchema{
					Attributes: []hclext.AttributeSchema{
						{Name: "source"},
					},
				},
			},
		},
	}, nil)
	if err != nil {
		return err
	}
	for _, module := range content.Blocks {
		attribute, exists := module.Body.Attributes["source"]
		if !exists {
			continue
		}
		var extract string
		_ = runner.EvaluateExpr(attribute.Expr, &extract, nil)
		{
			err := runner.EmitIssue(
				r,
				fmt.Sprintf("%s", extract),
				attribute.Expr.Range(),
			)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
