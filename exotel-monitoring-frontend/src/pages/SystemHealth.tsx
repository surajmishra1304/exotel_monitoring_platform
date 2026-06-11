import React, { useEffect, useState } from 'react';
import { Card, Row, Col, Typography, Space, Alert, Spin, Button, Divider, Switch, message } from 'antd';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import { useSystemHealth } from '@/hooks/useSystemHealth';
import { fmtRelative } from '@/utils/formatters';

const API_BASE = '/api/v1';

const { Title, Text } = Typography;

interface ServiceCardProps {
  name: string;
  status: 'CONNECTED' | 'DISCONNECTED';
  errorDetail?: string;
}

const ServiceCard: React.FC<ServiceCardProps> = ({ name, status, errorDetail }) => {
  const ok = status === 'CONNECTED';
  return (
    <Card
      style={{
        borderLeft: `4px solid ${ok ? '#52c41a' : '#ff4d4f'}`,
        borderRadius: 8,
      }}
    >
      <Space align="start" size={12}>
        {ok ? (
          <CheckCircleOutlined style={{ fontSize: 28, color: '#52c41a' }} />
        ) : (
          <CloseCircleOutlined style={{ fontSize: 28, color: '#ff4d4f' }} />
        )}
        <div>
          <Text strong style={{ fontSize: 16 }}>
            {name}
          </Text>
          <br />
          <Text
            style={{
              color: ok ? '#52c41a' : '#ff4d4f',
              fontWeight: 600,
              letterSpacing: 1,
            }}
          >
            {status}
          </Text>
          {errorDetail && (
            <>
              <br />
              <Text type="danger" style={{ fontSize: 12, wordBreak: 'break-word' }}>
                {errorDetail}
              </Text>
            </>
          )}
        </div>
      </Space>
    </Card>
  );
};

const SystemHealth: React.FC = () => {
  const { data, isLoading, dataUpdatedAt, refetch, isFetching } = useSystemHealth();

  const [skipCallLogsWrite, setSkipCallLogsWrite] = useState<boolean>(true);
  const [flagLoading, setFlagLoading] = useState(false);

  useEffect(() => {
    fetch(`${API_BASE}/settings`)
      .then(r => r.json())
      .then((d: { skip_call_logs_write: boolean }) => setSkipCallLogsWrite(d.skip_call_logs_write))
      .catch(() => {/* silently ignore on load */});
  }, []);

  const handleToggle = async (checked: boolean) => {
    setFlagLoading(true);
    try {
      const res = await fetch(`${API_BASE}/settings`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ skip_call_logs_write: checked }),
      });
      const d = await res.json();
      setSkipCallLogsWrite(d.skip_call_logs_write);
      message.success(`Skip call-logs write ${d.skip_call_logs_write ? 'enabled' : 'disabled'}`);
    } catch {
      message.error('Failed to update setting');
    } finally {
      setFlagLoading(false);
    }
  };

  if (isLoading) {
    return (
      <div style={{ textAlign: 'center', padding: 64 }}>
        <Spin size="large" />
        <br />
        <Text type="secondary" style={{ marginTop: 16, display: 'block' }}>
          Checking service health…
        </Text>
      </div>
    );
  }

  const isDegraded = data?.status === 'DEGRADED';

  return (
    <div style={{ maxWidth: 700 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <Title level={3} style={{ margin: 0 }}>
          System Health
        </Title>
        <Space>
          <Text type="secondary" style={{ fontSize: 12 }}>
            Last checked: {dataUpdatedAt ? fmtRelative(new Date(dataUpdatedAt).toISOString()) : '—'}
          </Text>
          <Button
            icon={<ReloadOutlined spin={isFetching} />}
            onClick={() => refetch()}
            size="small"
          >
            Refresh
          </Button>
        </Space>
      </div>

      {/* Overall status banner */}
      {isDegraded ? (
        <Alert
          type="error"
          showIcon
          message={<strong>Overall Status: DEGRADED</strong>}
          description="One or more infrastructure dependencies are unreachable. Data served by the platform may be stale or missing."
          style={{ marginBottom: 24 }}
        />
      ) : (
        <Alert
          type="success"
          showIcon
          message={<strong>Overall Status: UP</strong>}
          description="All systems are operational."
          style={{ marginBottom: 24 }}
        />
      )}

      <Divider orientation="left">Dependencies</Divider>

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12}>
          <ServiceCard
            name="MySQL"
            status={data?.mysql ?? 'DISCONNECTED'}
            errorDetail={data?.mysql_error}
          />
        </Col>
        <Col xs={24} sm={12}>
          <ServiceCard
            name="Redis"
            status={data?.redis ?? 'DISCONNECTED'}
            errorDetail={data?.redis_error}
          />
        </Col>
      </Row>

      <Divider orientation="left" style={{ marginTop: 32 }}>Feature Flags</Divider>

      <Card style={{ borderRadius: 8 }}>
        <Row align="middle" justify="space-between">
          <Col>
            <Text strong>Skip Call-Logs Write</Text>
            <br />
            <Text type="secondary" style={{ fontSize: 12 }}>
              When ON, individual call rows are not saved to <code>call_logs</code>. Snapshots are
              accumulated directly from the in-memory batch summary (faster, less DB I/O). Raw API
              responses are still preserved in <code>job_responses</code>.
            </Text>
          </Col>
          <Col>
            <Switch
              checked={skipCallLogsWrite}
              loading={flagLoading}
              onChange={handleToggle}
              checkedChildren="ON"
              unCheckedChildren="OFF"
            />
          </Col>
        </Row>
      </Card>

      <Divider />
      <Text type="secondary" style={{ fontSize: 12 }}>
        Auto-refreshes every 15 seconds. Click Refresh to check immediately.
      </Text>
    </div>
  );
};

export default SystemHealth;
