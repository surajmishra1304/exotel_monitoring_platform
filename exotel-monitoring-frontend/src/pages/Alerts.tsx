import React, { useState, useMemo } from 'react';
import {
  Table, Button, Select, Space, Typography, Tag, Row, Col, Card, Popconfirm,
} from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { useNavigate } from 'react-router-dom';
import { useActiveAlerts, useAcknowledgeAlert } from '@/hooks/useActiveAlerts';
import { useDashboardSummary } from '@/hooks/useDashboardSummary';
import { useExophones } from '@/hooks/useExophoneHealth';
import SeverityTag from '@/components/common/SeverityTag';
import EmptyState from '@/components/common/EmptyState';
import { fmtRelative } from '@/utils/formatters';
import type { Alert, Severity, AlertType } from '@/types/alert';

const { Title } = Typography;

const ALERT_TYPES: AlertType[] = [
  'CALL_DROP', 'API_FAILURE', 'HEARTBEAT_FAILURE', 'HIGH_LATENCY',
  'RETRY_EXHAUSTED', 'VENDOR_THROTTLING', 'EXOPHONE_DOWN', 'CACHE_FAILURE',
];

const Alerts: React.FC = () => {
  const navigate = useNavigate();
  const [severityFilter, setSeverityFilter] = useState<Severity | 'ALL'>('ALL');
  const [typeFilter, setTypeFilter] = useState<AlertType | 'ALL'>('ALL');

  const { data: alerts = [], isLoading, isError, error, refetch, isFetching } = useActiveAlerts();
  const { mutate: acknowledge, isPending } = useAcknowledgeAlert();

  // Build exophone number lookup across all accounts visible in dashboard.
  const { data: summary } = useDashboardSummary();
  const accountIds = (summary?.data ?? []).map((a) => a.account_id);
  // Fetch exophones for all accounts (first account only for now; extend as needed).
  const { data: page1 } = useExophones(accountIds[0] ?? 0, 1, 100);
  const { data: page2 } = useExophones(accountIds[1] ?? 0, 1, 100);
  const exophones1 = page1?.data ?? [];
  const exophones2 = page2?.data ?? [];
  const exophoneMap = useMemo(() => {
    const m: Record<number, string> = {};
    [...exophones1, ...exophones2].forEach((e) => { m[e.id] = e.exophone_number; });
    return m;
  }, [exophones1, exophones2]);

  if (isError) {
    return (
      <EmptyState
        description={`Failed to load alerts: ${(error as Error)?.message ?? 'Unknown error'}`}
        onRetry={() => refetch()}
      />
    );
  }

  const filtered = alerts.filter((a) => {
    if (severityFilter !== 'ALL' && a.severity !== severityFilter) return false;
    if (typeFilter !== 'ALL' && a.alert_type !== typeFilter) return false;
    return true;
  });

  const criticalCount = alerts.filter((a) => a.severity === 'CRITICAL').length;
  const warningCount  = alerts.filter((a) => a.severity === 'WARNING').length;
  const infoCount     = alerts.filter((a) => a.severity === 'INFO').length;

  const columns: ColumnsType<Alert> = [
    {
      title: 'Severity',
      dataIndex: 'severity',
      key: 'severity',
      width: 110,
      sorter: (a, b) => ({ CRITICAL: 0, WARNING: 1, INFO: 2 }[a.severity] ?? 3) - ({ CRITICAL: 0, WARNING: 1, INFO: 2 }[b.severity] ?? 3),
      defaultSortOrder: 'ascend',
      render: (s: Severity) => <SeverityTag severity={s} />,
    },
    {
      title: 'Type',
      dataIndex: 'alert_type',
      key: 'alert_type',
      render: (t: string) => <Tag>{t.replace(/_/g, ' ')}</Tag>,
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
    { title: 'Message', dataIndex: 'message', key: 'message', ellipsis: true },
    { title: 'Channel', dataIndex: 'channel', key: 'channel', width: 90 },
    {
      title: 'Triggered',
      dataIndex: 'triggered_at',
      key: 'triggered_at',
      width: 110,
      render: (v: string) => fmtRelative(v),
      sorter: (a, b) => new Date(b.triggered_at).getTime() - new Date(a.triggered_at).getTime(),
    },
    {
      title: 'Action',
      key: 'action',
      width: 110,
      render: (_: unknown, row: Alert) => (
        <Popconfirm title="Acknowledge this alert?" onConfirm={() => acknowledge(row.id)} okText="Yes">
          <Button size="small" loading={isPending}>Acknowledge</Button>
        </Popconfirm>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Title level={3} style={{ margin: 0 }}>Alerts Center</Title>
        <Button icon={<ReloadOutlined spin={isFetching} />} onClick={() => refetch()} size="small">Refresh</Button>
      </div>

      <Row gutter={[12, 12]} style={{ marginBottom: 16 }}>
        {[
          { label: 'CRITICAL', count: criticalCount, color: 'red' },
          { label: 'WARNING',  count: warningCount,  color: 'orange' },
          { label: 'INFO',     count: infoCount,     color: 'blue' },
        ].map(({ label, count, color }) => (
          <Col key={label}>
            <Card
              size="small"
              style={{ cursor: 'pointer', borderColor: severityFilter === label ? color : undefined }}
              onClick={() => setSeverityFilter(severityFilter === label ? 'ALL' : label as Severity)}
            >
              <Tag color={color}>{label}</Tag>
              <strong>{count}</strong>
            </Card>
          </Col>
        ))}
      </Row>

      <Space style={{ marginBottom: 16 }}>
        <Select
          value={severityFilter}
          onChange={setSeverityFilter}
          style={{ width: 140 }}
          options={[
            { value: 'ALL', label: 'All Severities' },
            { value: 'CRITICAL', label: 'CRITICAL' },
            { value: 'WARNING',  label: 'WARNING' },
            { value: 'INFO',     label: 'INFO' },
          ]}
        />
        <Select
          value={typeFilter}
          onChange={setTypeFilter}
          style={{ width: 200 }}
          options={[
            { value: 'ALL', label: 'All Types' },
            ...ALERT_TYPES.map((t) => ({ value: t, label: t.replace(/_/g, ' ') })),
          ]}
        />
      </Space>

      {!isLoading && filtered.length === 0 ? (
        <EmptyState description="No active alerts match the current filters." />
      ) : (
        <Table
          dataSource={filtered}
          columns={columns}
          rowKey="id"
          loading={isLoading}
          pagination={{ pageSize: 20 }}
          size="middle"
        />
      )}
    </div>
  );
};

export default Alerts;
