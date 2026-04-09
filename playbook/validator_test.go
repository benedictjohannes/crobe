package playbook

import (
	"strings"
	"testing"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    Playbook
		isAgent   bool
		wantError string
	}{
		{
			name: "Valid Config",
			config: Playbook{
				Title: "Test",
				Sections: []Section{
					{
						Title: "Section 1",
						Assertions: []Assertion{
							{Code: "A01", Title: "Assertion 1"},
						},
					},
				},
			},
			isAgent:   false,
			wantError: "",
		},
		{
			name: "Missing Code",
			config: Playbook{
				Title: "Test",
				Sections: []Section{
					{
						Title: "Section 1",
						Assertions: []Assertion{
							{Code: "", Title: "Missing Code"},
						},
					},
				},
			},
			isAgent:   false,
			wantError: "is missing a 'code'",
		},
		{
			name: "Duplicate Code",
			config: Playbook{
				Title: "Test",
				Sections: []Section{
					{
						Title: "Section 1",
						Assertions: []Assertion{
							{Code: "DUP", Title: "A1"},
							{Code: "DUP", Title: "A2"},
						},
					},
				},
			},
			isAgent:   false,
			wantError: "duplicate code found: DUP",
		},
		{
			name: "Agent Mode funcFile Error - PreCmd",
			config: Playbook{
				Title: "Test",
				Sections: []Section{
					{
						Title: "S1",
						Assertions: []Assertion{
							{
								Code: "P01",
								PreCmds: []Exec{
									{FuncFile: "test.ts"},
								},
							},
						},
					},
				},
			},
			isAgent:   true,
			wantError: "contains funcFile in preCmd",
		},
		{
			name: "Agent Mode funcFile Error - Cmd",
			config: Playbook{
				Title: "Test",
				Sections: []Section{
					{
						Title: "S1",
						Assertions: []Assertion{
							{
								Code: "C01",
								Cmds: []Cmd{
									{Exec: Exec{FuncFile: "test.ts"}},
								},
							},
						},
					},
				},
			},
			isAgent:   true,
			wantError: "contains funcFile in cmd",
		},
		{
			name: "Agent Mode funcFile Error - StdOutRule",
			config: Playbook{
				Title: "Test",
				Sections: []Section{
					{
						Title: "S1",
						Assertions: []Assertion{
							{
								Code: "R01",
								Cmds: []Cmd{
									{StdOutRule: EvaluationRule{FuncFile: "test.ts"}},
								},
							},
						},
					},
				},
			},
			isAgent:   true,
			wantError: "contains funcFile in stdOutRule",
		},
		{
			name: "Agent Mode funcFile Error - StdErrRule",
			config: Playbook{
				Title: "Test",
				Sections: []Section{
					{
						Title: "S1",
						Assertions: []Assertion{
							{
								Code: "R02",
								Cmds: []Cmd{
									{StdErrRule: EvaluationRule{FuncFile: "test.ts"}},
								},
							},
						},
					},
				},
			},
			isAgent:   true,
			wantError: "contains funcFile in stdErrRule",
		},
		{
			name: "Agent Mode funcFile Error - Gather",
			config: Playbook{
				Title: "Test",
				Sections: []Section{
					{
						Title: "S1",
						Assertions: []Assertion{
							{
								Code: "G01",
								Cmds: []Cmd{
									{
										Exec: Exec{
											Gather: []GatherSpec{
												{FuncFile: "test.ts"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			isAgent:   true,
			wantError: "contains funcFile in gather",
		},
		{
			name: "Agent Mode funcFile Error - PostCmd",
			config: Playbook{
				Title: "Test",
				Sections: []Section{
					{
						Title: "S1",
						Assertions: []Assertion{
							{
								Code: "P02",
								PostCmds: []Exec{
									{FuncFile: "test.ts"},
								},
							},
						},
					},
				},
			},
			isAgent:   true,
			wantError: "contains funcFile in postCmd",
		},
		{
			name: "Valid Agent Config",
			config: Playbook{
				Title: "Test",
				Sections: []Section{
					{
						Title: "Section 1",
						Assertions: []Assertion{
							{
								Code:  "A01",
								Title: "Assertion 1",
								PreCmds: []Exec{
									{Script: "echo 1"},
								},
								Cmds: []Cmd{
									{Exec: Exec{Script: "echo 2"}},
								},
								PostCmds: []Exec{
									{Script: "echo 3"},
								},
							},
						},
					},
				},
			},
			isAgent:   true,
			wantError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config, tt.isAgent)
			if tt.wantError == "" {
				if err != nil {
					t.Errorf("ValidateConfig() error = %v, want no error", err)
				}
			} else {
				if err == nil {
					t.Errorf("ValidateConfig() expected error containing %q, got nil", tt.wantError)
				} else if !strings.Contains(err.Error(), tt.wantError) {
					t.Errorf("ValidateConfig() error = %v, want error containing %q", err, tt.wantError)
				}
			}
		})
	}
}
