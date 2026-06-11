import React from 'react';
import {
  Row, Col, Table, Typography, Tag, Button, Tooltip, Space,
} from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { useNavigate } from 'react-router-dom';
import { useDashboardSummary } from '@/hooks/useDashboardSummary';
import KpiCard from '@/components/cards/KpiCard';
import EmptyState from '@/components/common/EmptyState';
import type { AccountDashboardSnapshot } from '@/types/snapshot';
import { fmtPercent, fmtLatency } from '@/utils/formatters';

const { Title, Text } = Typography;

const Dashboard: React.FC = () => {
  const navigate = useNavigate();
  const { data, isLoading, isFetching, isError, error, refetch } = useDashboardSummary();

  if (isError) {
    return (
      <EmptyState
        description={`Failed to load dashboard: ${(error as Error)?.message ?? 'Unknown error'}`}
        onRetry={() => refetch()}
      />
    );
  }

  const accounts: AccountDashboardSnapshot[] = data?.data ?? [];

  const totalCalls = accounts.reduce((s, a) => s + a.total_calls, 0);
  const totalFailed = accounts.reduce((s, a) => s + a.failed_calls, 0);
  const totalAlerts = accounts.reduce((s, a) => s + a.alert_count, 0);
  const avgSuccess =
    accounts.length > 0
      ? accounts.reduce((s, a) => s + a.success_rate, 0) / accounts.length
      : null;

  const columns: ColumnsType<AccountDashboardSnapshot> = [
    {
      title: 'Account',
      dataIndex: 'account_name',
      key: 'account_name',
      render: (name: string, row) => (
        <a onClick={() => navigate(`/exophones?account_id=${row.account_id}`)}>{name}</a>
      ),
    },
    {
      title: 'Exophones',
      key: 'exophones',
      render: (_, row) => `${row.active_exophones} / ${row.total_exophones}`,
    },
    { title: 'Total Calls', dataIndex: 'total_calls', key: 'total_calls', sorter: (a, b) => a.total_calls - b.total_calls },
    {
      title: 'Failed',
      dataIndex: 'failed_calls',
      key: 'failed_calls',
      render: (v: number) => <Tag color={v > 0 ? 'red' : 'default'}>{v}</Tag>,
    },
    {
      title: 'Success Rate',
      dataIndex: 'success_rate',
      key: 'success_rate',
      sorter: (a, b) => a.success_rate - b.success_rate,
      render: (v: number) => (
        <Text style={{ color: v >= 95 ? '#52c41a' : v >= 80 ? '#faad14' : '#ff4d4f' }}>
          {fmtPercent(v)}
        </Text>
      ),
    },
    {
      title: 'Avg Latency',
      dataIndex: 'avg_latency_ms',
      key: 'avg_latency_ms',
      render: (v: number | null) => fmtLatency(v),
    },
    {
      title: 'Alerts',
      dataIndex: 'alert_count',
      key: 'alert_count',
      render: (v: number) => <Tag color={v > 0 ? 'orange' : 'default'}>{v}</Tag>,
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <Title level={3} style={{ margin: 0 }}>Dashboard</Title>
        <Space>
          {data?.source && (
            <Text type="secondary" style={{ fontSize: 12 }}>
              Source: <Tag>{data.source}</Tag>
            </Text>
          )}
          <Button
            icon={<ReloadOutlined spin={isFetching} />}
            onClick={() => refetch()}
            size="small"
          >
            Refresh
          </Button>
        </Space>
      </div>

      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={12} sm={6}>
          <KpiCard title="Total Calls" value={totalCalls} loading={isLoading} />
        </Col>
        <Col xs={12} sm={6}>
          <KpiCard
            title="Failed Calls"
            value={totalFailed}
            loading={isLoading}
            highlight={totalFailed > 0 ? 'danger' : 'success'}
          />
        </Col>
        <Col xs={12} sm={6}>
          <KpiCard
            title="Avg Success Rate"
            value={avgSuccess != null ? `${avgSuccess.toFixed(1)}` : '—'}
            unit="%"
            loading={isLoading}
            highlight={
              avgSuccess == null ? undefined : avgSuccess >= 95 ? 'success' : avgSuccess >= 80 ? 'warning' : 'danger'
            }
          />
        </Col>
        <Col xs={12} sm={6}>
          <KpiCard
            title="Open Alerts"
            value={totalAlerts}
            loading={isLoading}
            highlight={totalAlerts > 0 ? 'warning' : 'success'}
          />
        </Col>
      </Row>

      {!isLoading && accounts.length === 0 ? (
        <EmptyState
          description="No account data available. The snapshot worker may not have run yet."
          onRetry={() => refetch()}
        />
      ) : (
        <Tooltip title="Click an account name to view its exophones">
          <Table
            dataSource={accounts}
            columns={columns}
            rowKey="account_id"
            loading={isLoading}
            pagination={false}
            size="middle"
          />
        </Tooltip>
      )}
    </div>
  );
};

export default Dashboard;
