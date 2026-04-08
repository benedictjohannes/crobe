# Playbook Development 🛠️

This guide covers advanced techniques for creating and managing ComplianceProbe playbooks, specifically focusing on the **Builder** workflow and **TypeScript/JavaScript** integration.

---

## 🏗️ The Preprocessing Pipeline (`funcFile`)

While you can write simple shell scripts and regex directly in your YAML, complex logic is better managed in external files. 

### 🎭 Raw vs. Baked Playbooks

-   **Raw Playbook**: Used during development. It uses the `funcFile` property to point to external `.ts` or `.js` files. 
    -   *Compatibility*: Only the `compliance-probe-builder` can read these.
-   **Baked Playbook**: A single, portable YAML file where all external scripts have been transpiled, minified, and inlined into the `func` property.
    -   *Compatibility*: Both the `compliance-probe` (Agent) and the Builder can run these.

### 🚀 Workflow

1.  **Draft**: Write a "Raw" YAML playbook using `funcFile` paths.
2.  **Develop**: Use TypeScript (`.ts`) for external scripts to get full IDE support, type checking, and linting.
3.  **Bake**: Run the preprocessor using the Builder:
    ```bash
    ./compliance-probe-builder --preprocess --input raw-playbook.yaml --output playbook.yaml
    ```
4.  **Result**: The builder transpiles TS to JS, minifies the code, and replaces `funcFile` with the inline `func` string.

### 📝 Example: Raw Playbook
```yaml
# raw-playbook.yaml
assertions:
  - code: SECURE_SHELL
    title: "SSH Configuration Audit"
    cmds:
      - exec:
          funcFile: "./scripts/get_ssh_config.ts" 
        stdOutRule:
          funcFile: "./scripts/validate_ssh.ts"
```

---

## 📜 TypeScript Logic & Runtime

ComplianceProbe uses an embedded **[Goja](https://github.com/dop251/goja)** engine for execution. While the runtime operates on JS, the **Builder** leverages `esbuild` to support TypeScript during development.

### 🛡️ Sandbox Restrictions
- **No Node.js APIs**: You cannot use `fs`, `path`, `http`, etc.
- **No External Imports**: All logic must be self-contained. You can use `import type` for type safety, but runtime code must be in the file or bundled.
- **Side-Effect Free**: The logic should purely process inputs and return strings or numbers.

---

## 🖇️ TypeScript Type Definitions

To help you get started, this repository includes a [`func.d.ts`](../typescript-sdk/func.d.ts) file with all the necessary type definitions. You can use these to ensure your scripts match the expected signatures.

### Using the Type Definitions
In your `.ts` files, you use `export default` to define the entry point. Once you have the types installed (`npm install compliance-probe`), you can import them directly:

```typescript
import type { ScriptContext } from "compliance-probe/func";

/**
 * The default export must be the function signature expected by the agent.
 */
export default ({ os }: ScriptContext): string => {
  return os === 'windows' ? 'dir' : 'ls -la';
};
```

### Core Type Signatures

The agent expects the transpiled file to result in a function. Using `export default` ensures the preprocessor captures your logic correctly.

#### 1. Dynamic Script Generation (`Exec.Func`)
Generates the shell command to run based on the current environment.

```typescript
import type { ScriptContext } from "compliance-probe/func";

export default ({ assertionContext, os, env }: ScriptContext): string => {
  if (os === 'windows') {
    return "powershell -Command Get-Service";
  }
  return "systemctl list-units";
};
```

#### 2. Evaluation Rules (`EvaluationRule.Func`)
Determines if a command passed or failed.

```typescript
import type { Evaluator } from "compliance-probe/func";

export default (stdout: string, stderr: string, context: any): -1 | 0 | 1 => {
  if (stderr.includes("error")) return -1;
  return stdout.length > 0 ? 1 : 0;
};
```

#### 3. Data Gathering (`GatherSpec.Func`)
Extracts specific values from command output to store in the `assertionContext`.

```typescript
import type { Gatherer } from "compliance-probe/func";

export default (stdout: string, stderr: string, context: any): string => {
  const match = stdout.match(/Version: ([\d.]+)/);
  return match ? match[1] : "unknown";
};
```

---

## 🛠️ Builder Commands Summary

- **Generate Schema**: Create `playbook.schema.json` for VS Code autocompletion.
  ```bash
  ./compliance-probe-builder --schema > playbook.schema.json
  ```
- **Preprocess**: Transform a development ("raw") playbook into a production-ready ("baked") one.
  ```bash
  ./compliance-probe-builder --preprocess --input <input> --output <output>
  ```
- **Test Run (Development)**: You can run a raw playbook directly using the builder without baking it first:
  ```bash
  ./compliance-probe-builder raw-playbook.yaml
  ```
