# CloudfixLinter Ruleset Template

## Requirements

- TFLint v0.35+
- Go v1.18

## Installation

You can install the plugin with `tflint --init`. Declare a config in `.tflint.hcl` as follows:

```hcl
plugin "template" {
  enabled = true

  version = "0.2.4"
  source  = "github.com/trilogy-group/tflint-ruleset-template"

}
```

## Building the plugin for local development

Clone the repository locally and run the following command:

```
$ make
```

You can easily install the built plugin with the following:

```
$ make install
```

The plugin can be run by executing `tflint` on the commmand line. More details about the supported commands can be found [here](https://github.com/terraform-linters/tflint/blob/master/README.md#usage)

This tool is meant to be used in conjunction with `cloudfix-linter`. Details about it can be found [here](https://github.com/trilogy-group/cloudfix-linter)

## Releasing a new version

1) Push a new tag
2( run `goreleaser release`)