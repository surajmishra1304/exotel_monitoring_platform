export interface Exophone {
  id: number;
  account_id: number;
  exophone_number: string;
  status: 'active' | 'inactive';
  priority: 'P0' | 'P1' | 'P2' | 'P3';
  monitoring_flag: number;
  is_cron_applicable: number;
  cache_ttl: number;
  metadata_json?: string;
  last_synced_at: string | null;
  created_at: string;
  updated_at: string;
}
