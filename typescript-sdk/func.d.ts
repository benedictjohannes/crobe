/**
 * Context within an assertion
 */
export interface AssertionContext {
  [key: string]: string;
}

export interface ScriptContext {
  /**
   * Key-value pairs gathered in the current assertion.
   * Scoped strictly to the assertion.
   */
  assertionContext: AssertionContext;
  
  /**
   * Environment variables available to the agent process.
   */
  env: Record<string, string>;
  
  /**
   * The operating system of the target machine.
   */
  os: 'linux' | 'windows' | 'mac';
  
  /**
   * The CPU architecture of the target machine.
   */
  arch: 'amd64' | 'arm64';
  
  /**
   * The username executing the agent.
   */
  user: string;
  
  /**
   * Current working directory of the agent.
   */
  cwd: string;
}

/**
 * Signature for Exec.Func
 * Generates the shell command or script to be executed.
 */
export type ScriptGenerator = (context: ScriptContext) => string;

/**
 * Signature for EvaluationRule.Func
 * Evaluates command output to determine a score.
 * @returns -1 (Fail), 0 (Neutral), 1 (Pass)
 */
export type Evaluator = (
  stdout: string,
  stderr: string,
  assertionContext: AssertionContext
) => -1 | 0 | 1;

/**
 * Signature for GatherSpec.Func
 * Extracts data from output to store in assertionContext[key].
 */
export type Gatherer = (
  stdout: string,
  stderr: string,
  assertionContext: AssertionContext
) => string;
