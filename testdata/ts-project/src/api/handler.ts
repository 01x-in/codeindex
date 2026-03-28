import { formatDate, parseId } from '../utils';

export interface RequestConfig {
  timeout: number;
  retries: number;
}

export type ResponsePayload = {
  data: unknown;
  timestamp: string;
};

export function handleRequest(id: string): ResponsePayload {
  const numId = parseId(id);
  const now = formatDate(new Date());
  return { data: { id: numId }, timestamp: now };
}

function validateHeaders(headers: Record<string, string>): boolean {
  return 'authorization' in headers;
}
