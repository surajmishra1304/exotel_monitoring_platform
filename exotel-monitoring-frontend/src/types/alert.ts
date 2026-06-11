export type AlertType =
  | 'API_FAILURE'
  | 'HEARTBEAT_FAILURE'
  | 'HIGH_LATENCY'
  | 'RETRY_EXHAUSTED'
  | 'VENDOR_THROTTLING'
  | 'EXOPHONE_DOWN'
  | 'CACHE_FAILURE'
  | 'CALL_DROP';

export type Severity = 'CRITICAL' | 'WARNING' | 'INFO';
export type AlertStatus = 'OPEN' | 'ACKNOWLEDGED' | 'RESOLVED';
export type AlertChannel = 'SLACK' | 'EMAIL' | 'WEBHOOK';

export interface Alert {
  id: number;
  transaction_id: string;
  exophone_id: number;
  account_id: number;
  alert_type: AlertType;
  severity: Severity;
  alert_status: AlertStatus;
  message: string;
  channel: AlertChannel;
  triggered_at: string;
  acknowledged_at: string | null;
  resolved_at: string | null;
  created_at: string;
  updated_at: string;
}
