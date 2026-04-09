package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/benedictjohannes/crobe/director"
	"github.com/benedictjohannes/crobe/internal/configsource"
	"github.com/benedictjohannes/crobe/internal/headerflags"
	"github.com/benedictjohannes/crobe/internal/reportwriter"
	"github.com/benedictjohannes/crobe/internal/transpile"
	"github.com/benedictjohannes/crobe/playbook"
	"github.com/benedictjohannes/crobe/report"
)

func main() {
	schemaFlag := flag.Bool("schema", false, "Output the configuration JSON schema and exit")
	preprocessFlag := flag.Bool("preprocess", false, "Preprocess a raw YAML into a baked playbook")
	inputFlag := flag.String("input", "", "Input raw YAML file (for preprocess)")
	outputFlag := flag.String("output", "playbook.yaml", "Output baked YAML file (for preprocess)")
	folderFlag := flag.String("folder", "", "Folder to write reports to (default \"reports\")")
	var headersFlags headerflags.HeaderFlags
	flag.Var(&headersFlags, "H", "Custom header for remote playbook fetching (eg: 'Authorization: Bearer <TOKEN>'). Specify multiple times for each header you want to add.")
	flag.Parse()

	headers := headersFlags.ToMap()
	reportwriter.DefaultReportsDir = *folderFlag

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
	configPath := flag.Arg(0)
	if configPath == "" {
		fmt.Println("❌ Error: No playbook provided. Use 'crobe [path/to/playbook.yaml]'")
		os.Exit(1)
	}

	config, _, err := configsource.LoadConfig(configPath, headers)
	if err != nil {
		fmt.Printf("❌ Failed to load playbook %s: %v\n", configPath, err)
		os.Exit(1)
	}

	// Validate (builder allows funcFile)
	if err := playbook.ValidateConfig(*config, false); err != nil {
		fmt.Printf("❌ Validation Error: %v\n", err)
		os.Exit(1)
	}

	// Transpile in-memory for direct run
	if err := transpile.Preprocess(config, filepath.Dir(configPath)); err != nil {
		fmt.Printf("❌ Preprocessing Error: %v\n", err)
		os.Exit(1)
	}

	trace := director.Run(*config)
	result := report.GenerateReport(trace)
	if err := reportwriter.DispatchReport(config, result); err != nil {
		fmt.Printf("❌ Reporting Error: %v\n", err)
		os.Exit(1)
	}

	if result.Structured.Stats.Failed > 0 {
		os.Exit(1)
	}
}

// Preprocess Logic

func runPreprocess(inputPath string, outputPath string) {
	if err := transpile.BakeFile(inputPath, outputPath); err != nil {
		fmt.Printf("❌ Preprocessing Failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("🚀 Preprocessing Complete! Baked playbook saved to: %s\n", outputPath)
}
