package main

import (
	"compliance-probe/executor"
	"compliance-probe/playbook"
	"compliance-probe/report"
	"compliance-probe/internal/reportwriter"
	"compliance-probe/internal/transpile"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func main() {
	schemaFlag := flag.Bool("schema", false, "Output the configuration JSON schema and exit")
	preprocessFlag := flag.Bool("preprocess", false, "Preprocess a raw YAML into a baked playbook")
	inputFlag := flag.String("input", "", "Input raw YAML file (for preprocess)")
	outputFlag := flag.String("output", "playbook.yaml", "Output baked YAML file (for preprocess)")
	flag.Parse()

	if *schemaFlag {
		schema, err := playbook.GenerateSchema()
		if err != nil {
			fmt.Printf("❌ Failed to generate schema: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(schema)
		return
	}

	if *preprocessFlag {
		if *inputFlag == "" {
			fmt.Println("❌ Error: --input is required for --preprocess")
			os.Exit(1)
		}
		runPreprocess(*inputFlag, *outputFlag)
		return
	}

	// Default: Run Agent Report
	configPath := getConfigPath()
	if configPath == "" {
		fmt.Println("❌ Error: No playbook provided. Use 'compliance-probe [path/to/playbook.yaml]'")
		os.Exit(1)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("❌ Failed to read playbook %s: %v\n", configPath, err)
		os.Exit(1)
	}

	var config playbook.ReportConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		fmt.Printf("❌ Failed to parse YAML: %v\n", err)
		os.Exit(1)
	}

	// Validate (builder allows funcFile)
	if err := playbook.ValidateConfig(config, false); err != nil {
		fmt.Printf("❌ Validation Error: %v\n", err)
		os.Exit(1)
	}

	result := report.GenerateReport(config, executor.RunExec)
	reportwriter.WriteToFolder(result)
}

func getConfigPath() string {
	args := flag.Args()
	if len(args) > 0 {
		return args[0]
	}
	if fileExists("playbook.yaml") {
		return "playbook.yaml"
	}
	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// Preprocess Logic

func runPreprocess(inputPath string, outputPath string) {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Printf("❌ Failed to read input: %v\n", err)
		os.Exit(1)
	}

	var config playbook.ReportConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		fmt.Printf("❌ Failed to parse YAML: %v\n", err)
		os.Exit(1)
	}

	// Walk and transpile
	for i := range config.Sections {
		for j := range config.Sections[i].Assertions {
			a := &config.Sections[i].Assertions[j]
			processAssertion(a, filepath.Dir(inputPath))
		}
	}

	// Validate
	if err := playbook.ValidateConfig(config, false); err != nil {
		fmt.Printf("❌ Validation Error: %v\n", err)
		os.Exit(1)
	}

	// Save "baked" playbook
	outData, err := yaml.Marshal(config)
	if err != nil {
		fmt.Printf("❌ Failed to marshal YAML: %v\n", err)
		os.Exit(1)
	}

	err = os.WriteFile(outputPath, outData, 0644)
	if err != nil {
		fmt.Printf("❌ Failed to write output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🚀 Preprocessing Complete! Baked playbook saved to: %s\n", outputPath)
}

func processAssertion(a *playbook.Assertion, baseDir string) {
	for i := range a.PreCmds {
		processExec(&a.PreCmds[i], baseDir)
	}
	for i := range a.Cmds {
		processExec(&a.Cmds[i].Exec, baseDir)
		processEvalRule(&a.Cmds[i].StdOutRule, baseDir)
		processEvalRule(&a.Cmds[i].StdErrRule, baseDir)
	}
	for i := range a.PostCmds {
		processExec(&a.PostCmds[i], baseDir)
	}
}

func processExec(e *playbook.Exec, baseDir string) {
	if e.FuncFile != "" {
		code, err := transpile.Transpile(filepath.Join(baseDir, e.FuncFile))
		if err != nil {
			fmt.Printf("❌ Transpilation Error (%s): %v\n", e.FuncFile, err)
			os.Exit(1)
		}
		e.Func = code
		e.FuncFile = ""
	}
	for i := range e.Gather {
		if e.Gather[i].FuncFile != "" {
			code, err := transpile.Transpile(filepath.Join(baseDir, e.Gather[i].FuncFile))
			if err != nil {
				fmt.Printf("❌ Transpilation Error (%s): %v\n", e.Gather[i].FuncFile, err)
				os.Exit(1)
			}
			e.Gather[i].Func = code
			e.Gather[i].FuncFile = ""
		}
	}
}

func processEvalRule(r *playbook.EvaluationRule, baseDir string) {
	if r.FuncFile != "" {
		code, err := transpile.Transpile(filepath.Join(baseDir, r.FuncFile))
		if err != nil {
			fmt.Printf("❌ Transpilation Error (%s): %v\n", r.FuncFile, err)
			os.Exit(1)
		}
		r.Func = code
		r.FuncFile = ""
	}
}

