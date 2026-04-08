# ComplianceProbe 🛡️

[![Build Status](https://img.shields.io/github/actions/workflow/status/benedictjohannes/ComplianceProbe/release.yml?style=flat-square)](https://github.com/benedictjohannes/ComplianceProbe/actions) [![Go Reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/benedictjohannes/ComplianceProbe) [![NPM Version](https://img.shields.io/npm/v/compliance-probe.svg?style=flat-square)](https://www.npmjs.com/package/compliance-probe) [![License: MIT](https://img.shields.io/github/license/benedictjohannes/ComplianceProbe?color=yellow&style=flat-square)](https://github.com/benedictjohannes/ComplianceProbe/blob/master/LICENSE)

**ComplianceProbe** is a cross-platform security compliance reporting agent. It executes a series of automated checks defined in a YAML "playbook" to verify system integrity, security configurations, and hardware state.

Whether you are auditing a desktop for security standards or monitoring server health, ComplianceProbe provides a flexible, scriptable, and reproducible way to generate detailed compliance reports.

## ✨ Key Features

-   **🔍 Automated Compliance Checks**: Group assertions into logical sections (e.g., OS Integrity, IAM, Data Protection).
-   **🚀 Multi-Platform support**: Native binaries for Linux, Windows, and macOS (Intel & ARM).
-   **📊 Comprehensive Reporting**: Generates reports in:
    -   **Markdown**: Human-readable summary for documentation.
    -   **JSON**: Machine-readable data for integration with other tools.
    -   **Detailed Logs**: Full execution trace for debugging.
-   **📥 Data Gathering**: Extract information from command outputs (via Regex or JS) and reuse it in subsequent checks within the same assertion.
-   **✅ Schema Validation**: Built-in JSON schema generation for IDE autocompletion.
-   **🌐 Remote Capabilities**: [Integrate playbook and compliance result submissions remotely](#remote-features).
-   **📜 JS Scripting & Logic**: Dynamic script generation and output evaluation using an embedded JavaScript engine ([Goja](https://github.com/dop251/goja)).
    -   **TS Support**: Write complex logic in separate `.js` or `.ts` files and "bake" them into a single portable playbook using the [builder tool](#builder-tool).
    -   **Type Definitions**: The [TypeScript definitions](./typescript-sdk) for playbook development and report consumption are available (via `npm install compliance-probe`).

## 🎯 Use Cases

-   **🌍 Adaptive Fleet Audits**: Run compliance checks across Linux, Windows, and macOS using a single **"Universal Playbook"** that adapts logic at runtime via JavaScript.
-   **🛡️ Dynamic Security Chaining**: Extract data (like current user or PID) in one step and use it to drive subsequent commands within the same assertion.
-   **🔐 Privacy-Aware Secret Validation**: Audit sensitive configurations for keys or PII without leaking them. Extract values for internal logic while explicitly excluding them from reports.
-   **📈 Weighted Compliance Scoring**: Assign scores to assertions to generate a numerical "Security Health" grade.
-   **🛠️ Pre-Flight Environment Checks**: Verify system integrity before deploying applications or onboarding new developer machines.

## 📦 Installation

Download the binary for your platform from the [releases](https://github.com/benedictjohannes/ComplianceProbe/releases) page:

-   `compliance-probe-linux`
-   `compliance-probe-windows.exe`
-   `compliance-probe-mac-arm`
-   `compliance-probe-mac-intel`

## 🚀 Quick Start

1.  **Run with the default playbook:**
    Ensure a `playbook.yaml` exists in the current directory and run the probe (replace with your platform binary):
    ```bash
    ./compliance-probe
    ```

2.  **Run with a specific playbook:**
    ```bash
    ./compliance-probe my-security-audit.yaml
    ```

3.  **View results:**
    Reports are saved to the directory specified by the `reportDestinationFolder` in the playbook, or the `--folder` CLI flag (which takes precedence). Defaults to `reports/`. Filenames are timestamped (e.g., `260206-033831.report.md`).

## 🛠️ Configuration (playbook.yaml)

The playbook defines what to check, how to score results, and how to extract data.

For a comprehensive guide on all available features—including **weighted scoring**, **embedded JavaScript logic**, **data gathering**, and **cross-platform handling**—see:

👉 **[playbook.example.yaml](./playbook.example.yaml)**

### Remote Features

The probe can integrate with a central compliance hub:
- Fetch playbooks from remote HTTPS URL
- Submit signed results via HTTPS POST to central compliance hub

👉 **[Remote Playbook & Submission Guide](./docs/RemotePlaybookSubmission.md)**

## Builder Tool

The **Builder** (`compliance-probe-builder`) is designed for compliance designers and developers to assist in creating and managing complex playbooks.

-   **Generate Schema**: Create a JSON schema for IDE autocompletion (VS Code, etc).
-   **Preprocessing Pipeline**: Use `funcFile` to externalize logic into TypeScript files, which are then transpiled and "baked" into a portable playbook for the agent.

For a detailed guide on using **TypeScript**, external scripts, and the preprocessing pipeline, see:

👉 **[Playbook Development Guide](./docs/PlaybookDevelopment.md)**

## 🏗️ Development and Building

The project is split into two packages under `cmd/` to separate the runtime agent from the developer tools.

### Prerequisites

- [Go](https://go.dev/dl/) 1.24+

### Build Agent Binaries

The agent is located in `cmd/probe`. To build optimized binaries for all platforms:

```bash
make build
```

Or build manually:
```bash
go build -o compliance-probe ./cmd/probe
```

### Build Builder Binaries

The builder is located in `cmd/builder`. It includes `esbuild` for TypeScript transpilation:

```bash
make build-builder
```

Or build manually:
```bash
go build -o compliance-probe-builder ./cmd/builder
```

### Running Tests

```bash
make test
```

## ⚖️ License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
