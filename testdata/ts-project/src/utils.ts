export function formatDate(date: Date): string {
  return date.toISOString();
}

export function parseId(raw: string): number {
  return parseInt(raw, 10);
}

export type Config = {
  port: number;
  host: string;
};

export interface Logger {
  info(msg: string): void;
  error(msg: string): void;
}
