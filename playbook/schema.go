package playbook

import (
	"encoding/json"
	"fmt"

	"github.com/invopop/jsonschema"
)

func GenerateSchema() (string, error) {
	reflector := &jsonschema.Reflector{
		DoNotReference: false,
		ExpandedStruct: true,
	}

	s := reflector.Reflect(&Playbook{})
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to generate schema: %v", err)
	}
	return string(data), nil
}
