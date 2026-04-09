package playbook

import (
	"fmt"
)

func ValidateConfig(config Playbook, isAgent bool) error {
	codes := make(map[string]bool)

	for _, section := range config.Sections {
		for _, assertion := range section.Assertions {
			if assertion.Code == "" {
				return fmt.Errorf("assertion '%s' in section '%s' is missing a 'code'", assertion.Title, section.Title)
			}
			if codes[assertion.Code] {
				return fmt.Errorf("duplicate code found: %s", assertion.Code)
			}
			codes[assertion.Code] = true

			if isAgent {
				if err := checkNoFuncFile(assertion); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func checkNoFuncFile(assertion Assertion) error {
	for _, exec := range assertion.PreCmds {
		if exec.FuncFile != "" {
			return fmt.Errorf("agent error: assertion %s contains funcFile in preCmd", assertion.Code)
		}
	}
	for _, cmd := range assertion.Cmds {
		if cmd.Exec.FuncFile != "" {
			return fmt.Errorf("agent error: assertion %s contains funcFile in cmd", assertion.Code)
		}
		if cmd.StdOutRule.FuncFile != "" {
			return fmt.Errorf("agent error: assertion %s contains funcFile in stdOutRule", assertion.Code)
		}
		if cmd.StdErrRule.FuncFile != "" {
			return fmt.Errorf("agent error: assertion %s contains funcFile in stdErrRule", assertion.Code)
		}
		for _, g := range cmd.Exec.Gather {
			if g.FuncFile != "" {
				return fmt.Errorf("agent error: assertion %s contains funcFile in gather", assertion.Code)
			}
		}
	}
	for _, exec := range assertion.PostCmds {
		if exec.FuncFile != "" {
			return fmt.Errorf("agent error: assertion %s contains funcFile in postCmd", assertion.Code)
		}
	}
	return nil
}
