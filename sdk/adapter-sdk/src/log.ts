export type LogFields = Record<string, unknown>;

/** Where the adapter writes its logs. Fields that look like credentials are redacted. */
export interface Logger {
  debug(msg: string, fields?: LogFields): void;
  info(msg: string, fields?: LogFields): void;
  warn(msg: string, fields?: LogFields): void;
  error(msg: string, fields?: LogFields): void;
}

const LEVELS = { debug: 10, info: 20, warn: 30, error: 40 } as const;
export type LogLevel = keyof typeof LEVELS;

const SECRET_KEY = /secret|token|password|passwd|authorization|api[_-]?key|credential|signature|cookie/i;

/** Replaces values whose key looks like a credential, at any depth. */
export function redact(v: unknown, depth = 0): unknown {
  if (depth > 6 || v === null || typeof v !== "object") return v;
  if (Array.isArray(v)) return v.map((x) => redact(x, depth + 1));
  const out: Record<string, unknown> = {};
  for (const [k, x] of Object.entries(v as Record<string, unknown>)) {
    out[k] = SECRET_KEY.test(k) ? "••••" : redact(x, depth + 1);
  }
  return out;
}

/** JSON lines on stderr, at or above level (REPUTECAST_ADAPTER_LOG_LEVEL, default info). */
export function consoleLogger(level: LogLevel = (process.env["REPUTECAST_ADAPTER_LOG_LEVEL"] as LogLevel | undefined) ?? "info"): Logger {
  const min = LEVELS[level] ?? LEVELS.info;
  const write = (lvl: LogLevel, msg: string, fields?: LogFields) => {
    if (LEVELS[lvl] < min) return;
    const line = { time: new Date().toISOString(), level: lvl, msg, ...((redact(fields ?? {}) as LogFields) ?? {}) };
    process.stderr.write(JSON.stringify(line) + "\n");
  };
  return {
    debug: (m, f) => write("debug", m, f),
    info: (m, f) => write("info", m, f),
    warn: (m, f) => write("warn", m, f),
    error: (m, f) => write("error", m, f),
  };
}

/** A logger that writes nothing. */
export const silentLogger: Logger = { debug() {}, info() {}, warn() {}, error() {} };
