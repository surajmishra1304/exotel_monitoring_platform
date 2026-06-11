import type { TxnStatus } from '@/types/transaction';
import type { Severity } from '@/types/alert';
import type { HeartbeatStatus } from '@/types/snapshot';

export const TXN_STATUS_COLOR: Record<TxnStatus, string> = {
  SUCCESS: 'success',
  RUNNING: 'processing',
  FAILED: 'error',
  RETRYING: 'warning',
  TIMEOUT: 'default',
};

export const SEVERITY_COLOR: Record<Severity, string> = {
  CRITICAL: 'red',
  WARNING: 'orange',
  INFO: 'blue',
};

export const HEARTBEAT_COLOR: Record<HeartbeatStatus, string> = {
  OK: '#52c41a',
  DEGRADED: '#faad14',
  OUTAGE: '#ff4d4f',
  UNKNOWN: '#d9d9d9',
};

export const HEARTBEAT_LABEL: Record<HeartbeatStatus, string> = {
  OK: 'OK',
  DEGRADED: 'DEGRADED',
  OUTAGE: 'OUTAGE',
  UNKNOWN: 'UNKNOWN',
};

export const PRIORITY_COLOR: Record<string, string> = {
  P0: 'red',
  P1: 'orange',
  P2: 'blue',
  P3: 'default',
};

export const CALL_CHART_COLORS = {
  connected: '#52c41a',
  dropped: '#ff7875',
  failed: '#ff4d4f',
  busy: '#faad14',
  missed: '#1890ff',
  canceled: '#bfbfbf',
  other: '#d3adf7',
};

export const LATENCY_WARN_MS = 2000;
export const LATENCY_CRIT_MS = 4000;
