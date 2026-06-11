export type TxnStatus = 'RUNNING' | 'SUCCESS' | 'FAILED' | 'RETRYING' | 'TIMEOUT';

export interface JobTransaction {
  id: number;
  transaction_id: string;
  job_id: number;
  account_id: number;
  exophone_id: number;
  job_type: string;
  status: TxnStatus;
  api_latency_ms: number | null;
  cache_hit: number;
  retry_count: number;
  error_message: string;
  started_at: string;
  completed_at: string | null;
  created_at: string;
  updated_at: string;
}
