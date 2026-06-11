import React, { useState, useMemo } from 'react';
import {
  Table, Select, Button, Space, Typography, DatePicker, Tag, Tooltip,
} from 'antd';
import { SearchOutlined, ReloadOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs, { Dayjs } from 'dayjs';
import { useNavigate } from 'react-router-dom';
import { useTransactions } from '@/hooks/useTransactions';
import { useDashboardSummary } from '@/hooks/useDashboardSummary';
import { useExophones } from '@/hooks/useExophoneHealth';
import type { TransactionFilters } from '@/api/transactions';
import StatusTag from '@/components/common/StatusTag';
import EmptyState from '@/components/common/EmptyState';
import { fmtDate, fmtLatency } from '@/utils/formatters';
import { LATENCY_WARN_MS, LATENCY_CRIT_MS } from '@/utils/constants';
import type { JobTransaction, TxnStatus } from '@/types/transaction';

const { Title, Text } = Typography;
const { RangePicker } = DatePicker;

const TXN_STATUSES: TxnStatus[] = ['RUNNING', 'SUCCESS', 'FAILED', 'RETRYING', 'TIMEOUT'];

const Transactions: React.FC = () => {
  const navigate = useNavigate();
  const [status, setStatus] = useState<TxnStatus | undefined>();
  const [exophoneIdFilter, setExophoneIdFilter] = useState<number | undefined>();
  const [dateRange, setDateRange] = useState<[Dayjs | null, Dayjs | null] | null>(null);
  const [limit, setLimit] = useState<number>(100);
  const [committed, setCommitted] = useState<TransactionFilters>({});

  const { data, isLoading, isError, error, refetch } = useTransactions(committed);

  // Build exophone number lookup.
  const { data: summary } = useDashboardSummary();
  const accountIds = (summary?.data ?? []).map((a) => a.account_id);
  const { data: page1 } = useExophones(accountIds[0] ?? 0);
  const { data: page2 } = useExophones(accountIds[1] ?? 0);
  const exophones1 = page1?.data ?? [];
  const exophones2 = page2?.data ?? [];
  const allExophones = useMemo(() => [...exophones1, ...exophones2], [exophones1, exophones2]);
  const exophoneMap = useMemo(() => {
    const m: Record<number, string> = {};
    allExophones.forEach((e) => { m[e.id] = e.exophone_number; });
    return m;
  }, [allExophones]);

  if (isError) {
    return (
      <EmptyState
        description={`Failed to load transactions: ${(error as Error)?.message ?? 'Unknown error'}`}
        onRetry={() => refetch()}
      />
    );
  }

  const handleSearch = () => {
    const filters: TransactionFilters = { limit };
    if (status) filters.status = status;
    if (exophoneIdFilter) filters.exophone_id = exophoneIdFilter;
    if (dateRange?.[0]) filters.from = dateRange[0].toISOString();
    if (dateRange?.[1]) filters.to = dateRange[1].toISOString();
    setCommitted(filters);
  };

  const latencyColor = (ms: number | null): string | undefined => {
    if (ms == null) return undefined;
    if (ms >= LATENCY_CRIT_MS) return '#ff4d4f';
    if (ms >= LATENCY_WARN_MS) return '#faad14';
    return undefined;
  };

  const columns: ColumnsType<JobTransaction> = [
    {
      title: 'Transaction ID',
      dataIndex: 'transaction_id',
      key: 'transaction_id',
      ellipsis: true,
      render: (v: string) => <Text code style={{ fontSize: 11 }}>{v}</Text>,
    },
    { title: 'Type', dataIndex: 'job_type', key: 'job_type', width: 110 },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (s: TxnStatus) => <StatusTag status={s} />,
    },
    {
      title: 'Exophone',
      dataIndex: 'exophone_id',
      key: 'exophone_id',
      render: (id: number) => (
        <a onClick={() => navigate(`/exophones/${id}`)}>
          {exophoneMap[id] ?? `#${id}`}
        </a>
      ),
    },
    {
      title: 'Latency',
      dataIndex: 'api_latency_ms',
      key: 'api_latency_ms',
      width: 100,
      render: (v: number | null) => (
        <span style={{ color: latencyColor(v) }}>{fmtLatency(v)}</span>
      ),
      sorter: (a, b) => (a.api_latency_ms ?? 0) - (b.api_latency_ms ?? 0),
    },
    {
      title: 'Retries',
      dataIndex: 'retry_count',
      key: 'retry_count',
      width: 80,
      render: (v: number) => v > 0 ? <Tag color="orange">{v}</Tag> : v,
    },
    {
      title: 'Cache Hit',
      dataIndex: 'cache_hit',
      key: 'cache_hit',
      width: 90,
      render: (v: number) => <Tag color={v ? 'green' : 'default'}>{v ? 'Yes' : 'No'}</Tag>,
    },
    {
      title: 'Started',
      dataIndex: 'started_at',
      key: 'started_at',
      width: 160,
      render: (v: string) => fmtDate(v),
      sorter: (a, b) => new Date(b.started_at).getTime() - new Date(a.started_at).getTime(),
      defaultSortOrder: 'ascend',
    },
  ];

  const expandedRowRender = (row: JobTransaction) => (
    <div style={{ padding: '8px 24px', background: '#fafafa', fontSize: 12 }}>
      <Space direction="vertical" size={2}>
        <Text><strong>Transaction ID:</strong> {row.transaction_id}</Text>
        <Text>
          <strong>Exophone:</strong> {exophoneMap[row.exophone_id] ?? `#${row.exophone_id}`}
          {' · '}<strong>Job ID:</strong> {row.job_id}
          {' · '}<strong>Account ID:</strong> {row.account_id}
        </Text>
        <Text><strong>Completed:</strong> {fmtDate(row.completed_at)}</Text>
        {row.error_message && (
          <Text type="danger"><strong>Error:</strong> {row.error_message}</Text>
        )}
      </Space>
    </div>
  );

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Title level={3} style={{ margin: 0 }}>Transaction Explorer</Title>
        <Tooltip title="Refresh with current filters">
          <Button icon={<ReloadOutlined />} onClick={() => refetch()} size="small" />
        </Tooltip>
      </div>

      {/* Filters */}
      <Space wrap style={{ marginBottom: 16 }}>
        <Select
          allowClear
          placeholder="Status"
          value={status}
          onChange={setStatus}
          style={{ width: 140 }}
          options={[
            { value: undefined, label: 'All Statuses' },
            ...TXN_STATUSES.map((s) => ({ value: s, label: s })),
          ]}
        />
        <Select
          allowClear
          showSearch
          placeholder="Exophone"
          value={exophoneIdFilter}
          onChange={setExophoneIdFilter}
          style={{ minWidth: 200 }}
          options={allExophones.map((e) => ({ value: e.id, label: e.exophone_number }))}
          filterOption={(input, opt) =>
            (opt?.label as string ?? '').toLowerCase().includes(input.toLowerCase())
          }
        />
        <RangePicker
          showTime
          value={dateRange}
          onChange={(v) => setDateRange(v as [Dayjs | null, Dayjs | null] | null)}
          presets={[
            { label: 'Last 1h',  value: [dayjs().subtract(1, 'hour'), dayjs()] },
            { label: 'Last 24h', value: [dayjs().subtract(24, 'hour'), dayjs()] },
            { label: 'Last 7d',  value: [dayjs().subtract(7, 'day'), dayjs()] },
          ]}
        />
        <Select
          value={limit}
          onChange={setLimit}
          style={{ width: 100 }}
          options={[
            { value: 50,  label: '50 rows' },
            { value: 100, label: '100 rows' },
            { value: 250, label: '250 rows' },
          ]}
        />
        <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>Search</Button>
      </Space>

      {!isLoading && (data?.data ?? []).length === 0 ? (
        <EmptyState description="No transactions found. Adjust filters and click Search." />
      ) : (
        <Table
          dataSource={data?.data ?? []}
          columns={columns}
          rowKey="id"
          loading={isLoading}
          pagination={{ pageSize: 50, showTotal: (t) => `${t} transactions` }}
          expandable={{ expandedRowRender }}
          size="small"
          scroll={{ x: 900 }}
        />
      )}
    </div>
  );
};

export default Transactions;
