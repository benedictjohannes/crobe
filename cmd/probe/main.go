package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/benedictjohannes/crobe/director"
	"github.com/benedictjohannes/crobe/internal/configsource"
	"github.com/benedictjohannes/crobe/internal/headerflags"
	"github.com/benedictjohannes/crobe/internal/reportwriter"
	"github.com/benedictjohannes/crobe/playbook"
	"github.com/benedictjohannes/crobe/report"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	flags := flag.NewFlagSet("crobe", flag.ContinueOnError)
	folderFlag := flags.String("folder", "", "Folder to write reports to (default \"reports\")")
	var headersFlags headerflags.HeaderFlags
	flags.Var(&headersFlags, "H", "Custom header for remote playbook fetching (eg: 'Authorization: Bearer <TOKEN>'). Specify multiple times for each header you want to add.")

	if err := flags.Parse(args); err != nil {
		return 1
	}

	headers := headersFlags.ToMap()

	reportwriter.DefaultReportsDir = *folderFlag

	configPath := flags.Arg(0)
	if configPath == "" {
		fmt.Println("❌ Error: No playbook provided. Use 'crobe [path/to/playbook.yaml]'")
		return 1
	}

	config, _, err := configsource.LoadConfig(configPath, headers)
	if err != nil {
		fmt.Printf("❌ Failed to load playbook %s: %v\n", configPath, err)
		return 1
	}

	// TODO this line to below are not covered by tests

	// Validate as Agent
	if err := playbook.ValidateConfig(*config, true); err != nil {
		fmt.Printf("❌ Validation Error: %v\n", err)
		return 1
	}

	trace := director.Run(*config)
	result := report.GenerateReport(trace)
	if err := reportwriter.DispatchReport(config, result); err != nil {
		fmt.Printf("❌ Reporting Error: %v\n", err)
		return 1
	}

	if result.Structured.Stats.Failed > 0 {
		return 1
	}

	return 0
}
