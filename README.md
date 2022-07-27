# TFLint Ruleset Template

## Requirements

- TFLint v0.35+
- Go v1.18

## Installation

You can install the plugin with `tflint --init`. Declare a config in `.tflint.hcl` as follows:

```hcl
plugin "template" {
  enabled = true

  version = "0.2.1"
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
