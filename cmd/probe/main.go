package main

import (
	"compliance-probe/executor"
	"compliance-probe/internal/configsource"
	"compliance-probe/internal/reportwriter"
	"compliance-probe/playbook"
	"compliance-probe/report"
	"flag"
	"fmt"
	"os"
)

func main() {
	flag.Parse()

	configPath := configsource.GetConfigSource(flag.Arg(0))
	if configPath == "" {
		fmt.Println("❌ Error: No playbook provided. Use 'compliance-probe [path/to/playbook.yaml]'")
		os.Exit(1)
	}

	config, _, err := configsource.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("❌ Failed to load playbook %s: %v\n", configPath, err)
		os.Exit(1)
	}

	// Validate as Agent
	if err := playbook.ValidateConfig(*config, true); err != nil {
		fmt.Printf("❌ Validation Error: %v\n", err)
		os.Exit(1)
	}

	result := report.GenerateReport(*config, executor.RunExec)
	if err := reportwriter.DispatchReport(config, result); err != nil {
		fmt.Printf("❌ Reporting Error: %v\n", err)
		os.Exit(1)
	}
}
