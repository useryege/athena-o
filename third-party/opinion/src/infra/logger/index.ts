export type LogLevel = 'trace' | 'debug' | 'info' | 'warn' | 'error';

export const logger = {
  info: (msg: string, data?: unknown) => log('info', msg, data),
  warn: (msg: string, data?: unknown) => log('warn', msg, data),
  error: (msg: string, data?: unknown) => log('error', msg, data),
  debug: (msg: string, data?: unknown) => log('debug', msg, data),
};

function log(level: LogLevel, msg: string, data?: unknown): void {
  const entry: Record<string, unknown> = {
    time: new Date().toISOString(),
    level,
    msg,
  };
  if (data !== undefined) entry['data'] = data;
  // eslint-disable-next-line no-console
  console.log(JSON.stringify(entry));
}
