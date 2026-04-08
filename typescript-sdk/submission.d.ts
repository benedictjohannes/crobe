/**
 * Represents a single assertion's execution result in the JSON report.
 * 
 * @template T The type of the context data gathered during execution. 
 * Defaults to a record of unknown values.
 */
export interface Assertion<T = Record<string, any>> {
  /** Timestamps for the assertion execution. */
  timestamps: {
    /** 
     * The start time of the assertion execution.
     * @format date-time (ISO 8601)
     * @example "2024-04-07T14:30:00Z"
     */
    start: string;
    /** 
     * The end time of the assertion execution.
     * @format date-time (ISO 8601)
     * @example "2024-04-07T14:30:05Z"
     */
    end: string;
  };

  /** Whether the assertion passed the required criteria. */
  passed: boolean;

  /** The actual score achieved by the assertion. */
  score: number;

  /** The minimum score required for the assertion to be considered passing. */
  minScore: number;

  /** 
   * Key-value pairs of data gathered during the execution.
   * This includes any 'gather' results that were not explicitly excluded from the report.
   */
  context: T;
}

/**
 * Summary statistics of the compliance run.
 */
export interface Stats {
  /** Total number of assertions that passed. */
  passed: number;
  /** Total number of assertions that failed. */
  failed: number;
}

/**
 * The final structured report generated after a playbook execution.
 * This matches the JSON output structure of ComplianceProbe.
 */
export interface FinalReport {
  /** Timestamps for the overall report generation. */
  timestamps: {
    /** 
     * The start time of the entire playbook run.
     * @format date-time (ISO 8601)
     * @example "2024-04-07T14:29:00Z"
     */
    start: string;
    /** 
     * The completion time of the entire playbook run.
     * @format date-time (ISO 8601)
     * @example "2024-04-07T14:31:00Z"
     */
    end: string;
  };

  /** The username of the user who executed the probe. */
  username: string;

  /** The operating system platform (e.g., "linux", "mac", "windows"). */
  os: string;

  /** The system architecture (e.g., "amd64", "arm64"). */
  arch: string;

  /** 
   * A map of assertion results, indexed by their unique assertion code.
   */
  assertions: Record<string, Assertion>;

  /** Aggregated pass/fail statistics. */
  stats: Stats;
}

/**
 * The outer "envelope" for remote report submissions when using 'format: json'.
 * This structure contains Base64 encoded versions of the report files and their signatures.
 */
export interface RemoteSubmission {
  /** 
   * Base64 encoded 'report.json'. 
   * This content matches the FinalReport schema.
   */
  json: string;
  /** 
   * HMAC-SHA256 signature of the 'json' Base64 string (if signatureSecret is set). 
   */
  jsonSignature?: string;

  /** Base64 encoded 'report.md' (human-readable markdown summary). */
  md: string;
  /** 
   * HMAC-SHA256 signature of the 'md' Base64 string (if signatureSecret is set). 
   */
  mdSignature?: string;

  /** Base64 encoded 'report.log' (execution trace). */
  log: string;
  /** 
   * HMAC-SHA256 signature of the 'log' Base64 string (if signatureSecret is set). 
   */
  logSignature?: string;
}
