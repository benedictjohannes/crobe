package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/benedictjohannes/crobe/internal/configsource"
	"github.com/benedictjohannes/crobe/internal/reportwriter"
	"github.com/benedictjohannes/crobe/playbook"
	"github.com/benedictjohannes/crobe/report"
)

func main() {
	folderFlag := flag.String("folder", "", "Folder to write reports to (default \"reports\")")
	flag.Parse()

	reportwriter.DefaultReportsDir = *folderFlag

	configPath := flag.Arg(0)
	if configPath == "" {
		fmt.Println("❌ Error: No playbook provided. Use 'crobe [path/to/playbook.yaml]'")
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

	result := report.GenerateReport(*config)
	if err := reportwriter.DispatchReport(config, result); err != nil {
		fmt.Printf("❌ Reporting Error: %v\n", err)
		os.Exit(1)
	}
}
