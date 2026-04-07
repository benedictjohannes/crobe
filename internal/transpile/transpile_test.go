package transpile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dop251/goja"
)

func TestTranspile(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "transpile-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name     string
		content  string
		wantSub  string
		isModule bool
	}{
		{
			name:    "simple typescript",
			content: "const x: number = 10; console.log(x);",
			wantSub: "const x=10;console.log(x);",
		},
		{
			name:    "module with export default",
			content: "export default () => { return 'hello'; };",
			wantSub: "var exports={};",
		},
		{
			name:    "module with named export",
			content: "export const hello = 'world';",
			wantSub: "return module.exports.default||module.exports",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tmpDir, tt.name+".ts")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			got, err := Transpile(path)
			if err != nil {
				t.Errorf("Transpile() error = %v", err)
				return
			}

			if !strings.Contains(got, tt.wantSub) {
				t.Errorf("Transpile() = %v, want to contain %v", got, tt.wantSub)
			}
		})
	}
}
func TestTranspileExecution(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "transpile-exec-*")
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "typescript function passing context",
			content: "export default ({ val }: { val: string }) => { return 'hello ' + val; };",
			want:    "hello world",
		},
		{
			name:    "simple expression return",
			content: "1 + 2",
			want:    "3",
		},
		{
			name:    "modern syntax",
			content: "const x = { a: 1 }; const y = { ...x, b: 2 }; y.a + y.b",
			want:    "3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tmpDir, tt.name+".ts")
			os.WriteFile(path, []byte(tt.content), 0644)

			got, err := Transpile(path)
			if err != nil {
				t.Fatalf("Transpile() error = %v", err)
			}

			// Execute in Goja
			vm := goja.New()
			val, err := vm.RunString(got)
			if err != nil {
				t.Fatalf("Goja error: %v", err)
			}

			var result string
			if fn, ok := goja.AssertFunction(val); ok {
				res, err := fn(goja.Undefined(), vm.ToValue(map[string]interface{}{"val": "world"}))
				if err != nil {
					t.Fatalf("Fn call error: %v", err)
				}
				result = res.String()
			} else {
				result = val.String()
			}

			if result != tt.want {
				t.Errorf("Result = %v, want %v", result, tt.want)
			}
		})
	}
}
