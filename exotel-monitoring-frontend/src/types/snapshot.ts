export interface CallMetricsSnapshot {
  id: number;
  snapshot_date: string;
  account_id: number;
  exophone_id: number;
  total_calls: number;
  // Leg 1 (A-leg): inbound call to VN triggering the flow (app-based trigger / panel Leg 1)
  leg1_total: number;
  leg1_drops: number;        // completed + ConversationDuration=0 + Leg2Status="" (customer abandoned in IVR)
  leg1_drop_rate: number;    // leg1_drops / leg1_total * 100
  // Leg 2 (B-leg): outbound call FROM VN to agent/CX (the actual conversation)
  leg2_total: number;
  leg2_drops: number;        // completed + ConversationDuration=0 on Leg2 CDR (customer didn't answer)
  // Status breakdown across all legs
  connected_calls: number;   // ConversationDuration > 0 (both parties actually bridged)
  dropped_calls: number;     // leg1_drops + leg2_drops (total drops across all legs)
  failed_calls: number;      // status=failed (network failure)
  no_answer_calls: number;   // status=no-answer (missed)
  busy_calls: number;        // status=busy
  canceled_calls: number;    // status=canceled
  other_calls: number;       // any unrecognised Exotel status
  avg_duration_sec: number;
  answer_rate: number;       // connected/total * 100
  drop_rate: number;         // leg2_drops / leg2_total * 100 (customer-side drop rate)
  success_rate: number;      // alias for answer_rate
  hourly_distribution: string; // JSON string: 24-element array [h0, h1, ..., h23]
  peak_hour: number;         // 0-23, hour with the most calls
  // Per-leg status breakdown
  leg1_no_answer: number;
  leg1_busy: number;
  leg1_failed: number;
  leg2_no_answer: number;
  leg2_busy: number;
  leg2_failed: number;
  leg2_canceled: number;
  // Exact Leg1Status → Leg2Status → count breakdown.
  // Stored as a JSON string in DB; parsed to object by the frontend.
  // Shape after parse: { "no-answer": { "canceled": 10713, "failed": 4081 }, "busy": { "canceled": 283 }, ... }
  leg_breakdown?: string | Record<string, Record<string, number>> | null;
  created_at: string;
  updated_at: string;
}

export type HeartbeatStatus = 'OK' | 'DEGRADED' | 'OUTAGE' | 'UNKNOWN';

export interface ExophoneHealthSnapshot {
  id: number;
  exophone_id: number;
  account_id: number;
  heartbeat_status: HeartbeatStatus;
  availability_percent: number;
  last_api_latency_ms: number | null;
  active_streams: number;
  last_checked_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface AccountDashboardSnapshot {
  account_id: number;
  account_name: string;
  total_exophones: number;
  active_exophones: number;
  total_calls: number;
  failed_calls: number;
  success_rate: number;
  avg_latency_ms: number | null;
  alert_count: number;
  snapshot_date: string;
}
