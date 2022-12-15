package rules

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/terraform-linters/tflint-plugin-sdk/hclext"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
)

// ReccomendationFlagRule flags of cloudifx reccommendations
type ReccomendationFlagRule struct {
	tflint.DefaultRule
	TagToID        map[string]map[string]string
	AttributeRecco map[string]map[string][]string
	Taggable       map[string]bool
	BlockLevels	   [][]string // BlockLevels store heirarchy of blocks. BlockLevels[0] > BlockLevels[1]
}

// Constructor for maaking the rule struct
func NewReccomendationFlagRule(tagIDMap map[string]map[string]string, reccoMap map[string]map[string][]string, taggableMap map[string]bool) *ReccomendationFlagRule {
	return &ReccomendationFlagRule{
		TagToID:        tagIDMap,
		AttributeRecco: reccoMap,
		Taggable:       taggableMap,
		BlockLevels: [][]string{},
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

// gives the list of attributes that the runner needs to extract and creates blocks heirarchy
func (r *ReccomendationFlagRule) getAttributeList() []string {
	var attributes []string
	for _, reccos := range r.AttributeRecco {
		for attribute := range reccos {
			if attribute == "NoAttributeMarker" {
				continue
			}
			// If attribute is nested within some block
			blocksList := strings.Split(attribute, ".")
			// Add blocks in heirarchy of nestedness
			for index, block := range blocksList {
				if index == (len(blocksList)-1) {
					break
				}
				if len(r.BlockLevels)<=index {
					r.BlockLevels = append(r.BlockLevels, []string{})
				}
				r.BlockLevels[index] = append(r.BlockLevels[index], block)
			}
			// Last element in blocksList is the attribute to search
			attributes = append(attributes, blocksList[len(blocksList)-1])
		}
	}
	attributes = append(attributes, "tags")
	return attributes
}

func (r *ReccomendationFlagRule) flagRecommendations(runner tflint.Runner, reccoforID map[string][]string, currentBlock *hclext.Block, blockName string) {
	for attributeType, attributeValue := range reccoforID {
		for _, recco := range attributeValue {
			// '$' is present at start if tagging needs to done at start of file
			if recco[0]=='$' {
				x := currentBlock.DefRange
				x.Start = hcl.Pos{Line: 1, Column: 1, Byte: 1}
				x.End = hcl.Pos{Line: 1, Column: 3, Byte: 1}
				runner.EmitIssue(
					r,
					fmt.Sprintf("%s: Description: \"%s\"", blockName, recco[1:]),
					x,
				)
				continue
			}
			if attributeType == "NoAttributeMarker" {
				runner.EmitIssue(
					r,
					fmt.Sprintf("%s: Description: \"%s\"", blockName, recco),
					currentBlock.DefRange,
				)
			} else {
				// search if block contains attribute
				attributeTerraform, existsAttribute := searchAttribute(attributeType, currentBlock)
				if !existsAttribute {
					// required attribute doesn't exists in block
					runner.EmitIssue(
						r,
						fmt.Sprintf("%s: Reduce cost by setting the value of attribute \"%s\" to \"%s\"", blockName, attributeType, recco),
						currentBlock.DefRange,
					)
					continue
				}
				// required attribute exists in block
				var extractAttribute string
				runner.EvaluateExpr(attributeTerraform.Expr, &extractAttribute, nil)
				if extractAttribute != recco {
					runner.EmitIssue(
						r,
						fmt.Sprintf("%s: Reduce cost by setting this value to \"%s\"", blockName, recco),
						attributeTerraform.Expr.Range(),
					)
				}
			}
		}
	}
}

func (r *ReccomendationFlagRule) getResourceMap(runner tflint.Runner, currentBlock *hclext.Block) (map[string]string, string, bool) {
	var blockName string = currentBlock.Type+" "+currentBlock.Labels[0]
	if currentBlock.Type=="resource" {
		blockName += " "+currentBlock.Labels[1]
	}
	// find tag variable in block
	tags, exists := currentBlock.Body.Attributes["tags"]
	if !exists {
		// resource doesn't has tag variable
		// check if currentblock is taggable
		_, ok := r.Taggable[currentBlock.Labels[0]]
		if ok {
			runner.EmitIssue(
				r,
				"This resources is missing tags. Fix by running \"cloudfix-linter addTags\" followed by \"terraform apply\"!",
				currentBlock.DefRange,
			)
		}
		return nil, blockName, true
	}
	// get map of all tags added
	var getTags map[string]string
	_ = runner.EvaluateExpr(tags.Expr, &getTags, nil)

	// find yor_tag
	var yor_trace, foundY = getTags["yor_trace"]
	if !foundY {
		runner.EmitIssue(
			r,
			"This resource is missing a trace id tag. Fix by running \"cloudfix-linter addTags\" followed by \"terraform apply\"!",
			tags.Expr.Range(),
		)
		return nil, blockName, true
	}
	yorTraceStrip := strings.Trim(yor_trace, "\n")
	yorTraceTrim := strings.Trim(yorTraceStrip, `"`)

	// find recommendations for yor_tag
	var resourceMap, foundA = r.TagToID[yorTraceTrim]
	if !foundA {
		runner.EmitIssue(
			r,
			fmt.Sprintf("Couldn't find a matching AWS resource: \"%s\".Either it hasn't been deployed, or the trace ID has been changed. Run \"terraform apply\"!", yorTraceTrim),
			tags.Expr.Range(),
		)
		return nil, blockName, true
	}
	return resourceMap, blockName, false
}

func (r *ReccomendationFlagRule) scanModules(runner tflint.Runner, modules *hclext.BodyContent) {
	// scan all modules in current file
	for _, module := range modules.Blocks {
		// get map of recommendations for current module
		resourceMap, module_name, flagged := r.getResourceMap(runner, module)
		if flagged {
			continue
		}
		// find all resources deployed by current module
		resourceIDs := []string{}
		for _, resourceID := range resourceMap {
			resourceIDs = append(resourceIDs, resourceID)
		}
		// emit issues for all resources
		for _, resourceID := range resourceIDs {
			resource_Strip := strings.Trim(resourceID, "\n")
			resourceTrim := strings.Trim(resource_Strip, `"`)
			reccoforID, present := r.AttributeRecco[resourceTrim]
			if present {
				r.flagRecommendations(runner, reccoforID, module, module_name)
			}
		}
	}
}

func (r *ReccomendationFlagRule) scanResources(runner tflint.Runner, resources *hclext.BodyContent) {
	// scan all resources in current file
	for _, resource := range resources.Blocks {
		// get map of recommendations for current resource
		resourceMap, resource_name, flagged := r.getResourceMap(runner, resource)
		if flagged {
			continue
		}
		// find reccomendations specific to current resource as there may be multiple resources for yor_tag
		resourceID, exists := resourceMap[resource.Labels[0]+"&"+resource.Labels[1]]
		if !exists {
			continue
		}
		resourceStrip := strings.Trim(resourceID, "\n")
		resourceTrim := strings.Trim(resourceStrip, `"`)
		reccoforID := r.AttributeRecco[resourceTrim]
		// emit issues for all recommendations
		r.flagRecommendations(runner, reccoforID, resource, resource_name)
	}
}

func findAttribute(blocksList []string, index int, currentBlock *hclext.Block)(*hclext.Attribute, bool) {
	if index == (len(blocksList)-1) {
		v, b := currentBlock.Body.Attributes[blocksList[index]]
		return v, b
	}
	blocksWithName := currentBlock.Body.Blocks.ByType()[blocksList[index]]
	if len(blocksWithName)==0 {
		return nil, false
	}
	return findAttribute(blocksList, index+1, blocksWithName[0])
}

func searchAttribute(attributeType string, currentBlock *hclext.Block) (*hclext.Attribute, bool) {
	// split attributeType into component blocks
	// last block would be the final attribute
	blocksList := strings.Split(attributeType, ".")
	return findAttribute(blocksList, 0, currentBlock)
}


func (r *ReccomendationFlagRule) resolveBlockRule(level int, schema []hclext.AttributeSchema) []hclext.BlockSchema{
	newSchema := []hclext.BlockSchema{}
	// if all blocks are scanned then return
	if level == len(r.BlockLevels) {
		return newSchema
	}
	// create block of all types in currentBlockLevel
	for _, blockName := range r.BlockLevels[level] {
		childBlockSchema := r.resolveBlockRule(level+1, schema)
		newSchema = append(newSchema, 
			hclext.BlockSchema{
				Type: blockName,
				Body: &hclext.BodySchema{
					Attributes: schema,
					Blocks: childBlockSchema,
				},
			},
		)
	}
	return newSchema
}

// check flags off cloudfix recommendations.
func (r *ReccomendationFlagRule) Check(runner tflint.Runner) error {
	// get all attributes that should be found in terraform file
	var attributes = r.getAttributeList()
	// schema stores name of attributes to search in terraform file
	var schema []hclext.AttributeSchema
	for _, attribute := range attributes {
		var temp hclext.AttributeSchema
		temp.Name = attribute
		schema = append(schema, temp)
	}
	// find all resources in current file
	resources, err := runner.GetModuleContent(&hclext.BodySchema{
		Blocks: []hclext.BlockSchema{
			{
				Type:       "resource",
				LabelNames: []string{"resource_type", "resource_name"},
				Body: &hclext.BodySchema{
					Attributes: schema,
					// recursively create heirarchy of blocks to search for
					Blocks: r.resolveBlockRule(0, schema),
				},
			},
		},
	}, nil)
	if err != nil {
		return err
	}
	// emit Issues for resources
	r.scanResources(runner, resources)
	// find all modules in current file
	modules, err := runner.GetModuleContent(&hclext.BodySchema{
		Blocks: []hclext.BlockSchema{
			{
				Type:       "module",
				LabelNames: []string{"local_name"},
				Body: &hclext.BodySchema{
					Attributes: schema,
				},
			},
		},
	}, nil)
	if err != nil {
		return err
	}
	// emit issues for modules
	r.scanModules(runner, modules)
	return nil
}
