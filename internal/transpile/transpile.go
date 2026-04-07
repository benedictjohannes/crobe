package transpile

import (
	"fmt"
	"os"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

// Transpile converts TypeScript/Modern JS to a format suitable for the Goja runtime.
func Transpile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	codeStr := string(data)
	isModule := false
	trimmed := strings.TrimSpace(codeStr)
	if (len(trimmed) >= 7 && trimmed[:7] == "export ") ||
		(len(trimmed) >= 7 && trimmed[:7] == "import ") ||
		strings.Contains(codeStr, "\nexport ") ||
		strings.Contains(codeStr, "\nimport ") {
		isModule = true
	}

	opts := api.TransformOptions{
		Loader: api.LoaderTS,
		Target: api.ES2022,
	}

	if isModule {
		opts.Format = api.FormatCommonJS
		opts.MinifyWhitespace = true
		opts.MinifyIdentifiers = true
		opts.MinifySyntax = true
	} else {
		opts.Format = api.FormatDefault
		opts.MinifyWhitespace = true
	}

	result := api.Transform(codeStr, opts)
	if len(result.Errors) > 0 {
		return "", fmt.Errorf("esbuild error: %v", result.Errors[0].Text)
	}

	if isModule {
		return fmt.Sprintf("(function(){var exports={};var module={exports:exports};%s;return module.exports.default||module.exports})()", result.Code), nil
	}

	return strings.TrimSpace(string(result.Code)), nil
}
