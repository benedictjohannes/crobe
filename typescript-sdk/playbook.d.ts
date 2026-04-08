import { AssertionContext } from './func';

/**
 * An individual check unit with its own Pass/Fail verdict.
 * Data gathered via 'gather' is stored in a local context (AssertionContext)
 * which is strictly scoped to this assertion and NOT shared with others.
 */
export interface Assertion {
  /**
   * Unique code for the assertion (e.g., 'AUTH-001').
   * Minimum 3 characters.
   */
  code: string;

  /**
   * Title of the assertion.
   * Minimum 3 characters.
   */
  title: string;

  /**
   * Detailed description of what is being checked.
   * Minimum 3 characters.
   */
  description: string;

  /**
   * Executions performed before the main commands.
   */
  preCmds?: Exec[];

  /**
   * Main command units to execute.
   * At least one command is required.
   */
  cmds: Cmd[];

  /**
   * Executions performed after the main commands.
   */
  postCmds?: Exec[];

  /**
   * Minimum score required to consider the assertion as passed.
   * Default is 1.
   */
  minPassingScore?: number;

  /**
   * Message shown if the assertion passes.
   * Minimum 3 characters.
   */
  passDescription: string;

  /**
   * Message shown if the assertion fails.
   * Minimum 3 characters.
   */
  failDescription: string;
}

/**
 * A single command unit with execution and evaluation rules.
 */
export interface Cmd {
  /**
   * Execution details for the command.
   */
  exec: Exec;

  /**
   * Score added if the command's evaluation passes.
   * Default is 1.
   */
  passScore?: number;

  /**
   * Score added if the command's evaluation fails.
   * Default is -1.
   */
  failScore?: number;

  /**
   * Rule for evaluating the command's standard output.
   */
  stdOutRule?: EvaluationRule;

  /**
   * Rule for evaluating the command's standard error.
   */
  stdErrRule?: EvaluationRule;

  /**
   * Rules for evaluating the command's exit code.
   */
  exitCodeRules?: ExitCodeRule[];
}

/**
 * Execution details, supporting shell scripts or embedded JS.
 */
export interface Exec {
  /**
   * Shell to use (e.g., 'bash', 'powershell', 'sh').
   */
  shell?: string;

  /**
   * Script to execute in the chosen shell.
   */
  script?: string;

  /**
   * Embedded JS code for dynamic execution logic.
   * Signature: ({ assertionContext, env, os, arch, user, cwd }) => string
   */
  func?: string;

  /**
   * Path to JS/TS file (BUILDER ONLY).
   * Using this in a real playbook will cause an error.
   */
  funcFile?: string;

  /**
   * Data extraction specifications from command output.
   */
  gather?: GatherSpec[];

  /**
   * If true, hides stdout/stderr results from logs and markdown reports.
   */
  excludeFromReport?: boolean;
}

/**
 * Rule for evaluating command output using regex or JS logic.
 */
export interface EvaluationRule {
  /**
   * Regex to match against the output.
   */
  regex?: string;

  /**
   * If true, includes stderr in the evaluation.
   * Default is false.
   */
  includeStdErr?: boolean;

  /**
   * JS function for custom evaluation.
   * Signature: (stdout, stderr, assertionContext) => -1 | 0 | 1
   */
  func?: string;

  /**
   * Path to JS/TS file (BUILDER ONLY).
   * Using this in a real playbook will cause an error.
   */
  funcFile?: string;
}

/**
 * Specification for extracting data from command output.
 */
export interface GatherSpec {
  /**
   * Key to store the extracted data in assertionContext.
   */
  key: string;

  /**
   * If true, hides the extracted key from the JSON report.
   */
  excludeFromReport?: boolean;

  /**
   * Regex with capture groups to extract data from output.
   */
  regex?: string;

  /**
   * If true, includes stderr in the extraction evaluation.
   * Default is false.
   */
  includeStdErr?: boolean;

  /**
   * JS function for custom extraction logic.
   * Signature: (stdout, stderr, assertionContext) => string
   */
  func?: string;

  /**
   * Path to JS/TS file (BUILDER ONLY).
   * Using this in a real playbook will cause an error.
   */
  funcFile?: string;
}

/**
 * Rule for evaluating exit codes.
 */
export interface ExitCodeRule {
  /**
   * Minimum exit code (inclusive).
   */
  min?: number;

  /**
   * Maximum exit code (inclusive).
   */
  max?: number;

  /**
   * Score result if the exit code falls within range:
   * - -1: Fail
   * -  0: Neutral
   * -  1: Pass
   */
  result: -1 | 0 | 1;
}

/**
 * Supported destinations for generating reports.
 */
export type ReportDestination = 'folder' | 'https';

/**
 * A group of assertions with a title and description.
 */
export interface Section {
  /**
   * Title of the section.
   * Minimum 3 characters.
   */
  title: string;

  /**
   * Group of descriptions for the section.
   * At least one description is required.
   */
  description: string[];

  /**
   * List of assertions within this section.
   * At least one assertion is required.
   */
  assertions: Assertion[];
}

/**
 * Supported formats for remote report submission.
 */
export type ReportFormat = 'multipart' | 'json';

/**
 * Configuration for submitting reports to an HTTPS endpoint.
 */
export interface ReportDestinationConfig {
  /**
   * URL to post the report content to.
   */
  url: string;

  /**
   * Format of the report submission.
   * Default is 'multipart'.
   */
  format?: ReportFormat;

  /**
   * Secret used for HMAC-SHA256 signing of the payload.
   */
  signatureSecret?: string;

  /**
   * Custom HTTP headers to include in the submission.
   */
  additionalHeaders?: Record<string, string>;
}

/**
 * Root configuration structure for a compliance playbook.
 * 
 * Core Concepts:
 * 1. Sections: Logical groups of assertions (e.g., "System Foundation", "Identity").
 * 2. Assertions: Individual check units with their own Pass/Fail verdict.
 */
export type Playbook = {
  /**
   * Title of the report.
   * Minimum 3 characters.
   */
  title: string;

  /**
   * Custom metadata for the report (YAML frontmatter).
   * These values are passed to the generated markdown report.
   * If not specified, the report's title and date are added automatically.
   */
  reportFrontmatter?: Record<string, any>;

  /**
   * Sections containing assertions.
   * At least one section is required.
   */
  sections: Section[];

  /**
   * Destination type for the report.
   * Default is 'folder'.
   */
  reportDestination?: ReportDestination;

  /**
   * Specific configuration for HTTPS submission.
   */
  reportDestinationHttps?: ReportDestinationConfig;
}
