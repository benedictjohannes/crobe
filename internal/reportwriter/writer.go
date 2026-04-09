package reportwriter

import (
	"fmt"

	"github.com/benedictjohannes/crobe/playbook"
	"github.com/benedictjohannes/crobe/report"
)

// DispatchReport decides where to send the report based on the configuration.
func DispatchReport(config *playbook.Playbook, res report.FinalResult) error {
	switch config.ReportDestination {
	case playbook.ReportDestinationHTTPS:
		if config.ReportDestinationHTTPS == nil {
			return fmt.Errorf("reportDestination is 'https' but reportDestinationHttps is missing")
		}
		return WriteToHTTP(config.ReportDestinationHTTPS, res)
	case playbook.ReportDestinationFolder, "":
		reportsDir := DefaultReportsDir
		if reportsDir == "" {
			reportsDir = config.ReportDestinationFolder
		}
		return WriteToFolder(reportsDir, res)
	default:
		return fmt.Errorf("unknown reportDestination: %s", config.ReportDestination)
	}
}
