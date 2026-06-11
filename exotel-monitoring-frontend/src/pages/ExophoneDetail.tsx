import React, { useState } from 'react';
import {
  Row, Col, Card, Typography, Space, Button, Descriptions, Popconfirm, Tag,
  DatePicker, Alert, message, Switch, Tooltip,
} from 'antd';
import { ArrowLeftOutlined, ReloadOutlined, ClockCircleOutlined, DownloadOutlined, SyncOutlined } from '@ant-design/icons';
import {
  PieChart, Pie, Cell, Tooltip as RechartsTooltip, Legend, ResponsiveContainer,
  BarChart, Bar, XAxis, YAxis, CartesianGrid,
} from 'recharts';
import dayjs, { type Dayjs } from 'dayjs';
import { useParams, useNavigate } from 'react-router-dom';
import { useExophoneMetrics, useExophoneCallAnalytics } from '@/hooks/useExophoneMetrics';
import { useActiveAlerts, useAcknowledgeAlert } from '@/hooks/useActiveAlerts';
import { useExophoneById } from '@/hooks/useExophoneById';
import { useDashboardSummary } from '@/hooks/useDashboardSummary';
import HeartbeatBadge from '@/components/cards/HeartbeatBadge';
import KpiCard from '@/components/cards/KpiCard';
import SeverityTag from '@/components/common/SeverityTag';
import LoadingSpinner from '@/components/common/LoadingSpinner';
import EmptyState from '@/components/common/EmptyState';
import { fmtRelative, fmtLatency, fmtPercent, fmtDuration } from '@/utils/formatters';
import { CALL_CHART_COLORS, PRIORITY_COLOR } from '@/utils/constants';
import type { Alert as AlertRecord } from '@/types/alert';

const { Title, Text } = Typography;

function parseHourlyDistribution(raw: string | undefined | null): number[] {
  if (!raw) return Array(24).fill(0);
  try {
    const arr = JSON.parse(raw);
    if (Array.isArray(arr) && arr.length === 24) return arr as number[];
  } catch {
    // fall through
  }
  return Array(24).fill(0);
}

function fmtHour(h: number): string {
  return h.toString().padStart(2, '0') + ':00';
}

const ExophoneDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const exophoneId = Number(id);

  const [selectedDate, setSelectedDate] = useState<Dayjs | null>(null);
  const [reprocessing, setReprocessing] = useState(false);
  const [skipCallLogs, setSkipCallLogs] = useState<boolean | null>(null);
  const [savingSkip, setSavingSkip] = useState(false);
  const dateStr = selectedDate ? selectedDate.format('YYYY-MM-DD') : undefined;

  const { data: exophone } = useExophoneById(exophoneId);
  const { data: summary } = useDashboardSummary();
  const { data: metrics, isLoading: loadingMetrics, refetch: refetchMetrics } = useExophoneMetrics(exophoneId);
  const { data: callMeta, isLoading: loadingCalls, refetch: refetchCalls } = useExophoneCallAnalytics(exophoneId, dateStr);
  const { data: allAlerts = [] } = useActiveAlerts();
  const { mutate: acknowledge, isPending } = useAcknowledgeAlert();

  const exophoneAlerts = allAlerts.filter((a: AlertRecord) => a.exophone_id === exophoneId);

  const accountName = summary?.data.find(
    (a) => a.account_id === (exophone?.account_id ?? metrics?.account_id),
  )?.account_name ?? `Account ${exophone?.account_id ?? metrics?.account_id ?? '—'}`;

  const displayNumber = exophone?.exophone_number ?? `Exophone #${exophoneId}`;

  const handleRefresh = () => { refetchMetrics(); refetchCalls(); };

  const accountID = exophone?.account_id ?? metrics?.account_id;
  const handleExportCSV = () => {
    if (!accountID) return;
    const date = selectedDate ? selectedDate.format('YYYY-MM-DD') : new Date().toISOString().slice(0, 10);
    const url = `/api/v1/accounts/${accountID}/performance/export?exophone_id=${exophoneId}&date=${date}`;
    const a = document.createElement('a');
    a.href = url;
    a.download = '';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
  };

  const handleReprocess = async () => {
    setReprocessing(true);
    const date = selectedDate ? selectedDate.format('YYYY-MM-DD') : new Date().toISOString().slice(0, 10);
    try {
      const res = await fetch(`/api/v1/exophones/${exophoneId}/snapshot/reprocess?date=${date}`, { method: 'POST' });
      const body = await res.json();
      if (!res.ok) {
        message.error(body.error ?? 'Reprocess failed');
      } else {
        message.success(
          `Reprocessed ${body.unique_call_records} unique calls → answer rate ${body.answer_rate_pct}%`
        );
        refetchCalls();
      }
    } catch {
      message.error('Network error during reprocess');
    } finally {
      setReprocessing(false);
    }
  };

  // Sync skipCallLogs from fetched exophone data (once).
  const currentSkip = exophone?.skip_call_logs === 1;
  const effectiveSkip = skipCallLogs !== null ? skipCallLogs : currentSkip;

  const handleToggleSkipCallLogs = async (checked: boolean) => {
    setSavingSkip(true);
    try {
      const res = await fetch(`/api/v1/exophones/${exophoneId}/skip-call-logs`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ skip_call_logs: checked ? 1 : 0 }),
      });
      const body = await res.json();
      if (!res.ok) {
        message.error(body.error ?? 'Failed to update skip_call_logs');
      } else {
        setSkipCallLogs(checked);
        message.success(checked ? 'Call logs will be skipped for this exophone' : 'Call logs will be written for this exophone');
      }
    } catch {
      message.error('Network error');
    } finally {
      setSavingSkip(false);
    }
  };

  const callData = callMeta?.data ?? null;
  const isStale = callMeta?.stale ?? false;
  const isPending2 = callMeta?.pending ?? false;
  const dataAsOf = callMeta?.dataAsOf ?? null;

  // IVR mode: all calls are Leg1 so leg2_total=0, but we still have leg1_drops and leg2_drops.
  // Show them as separate pie slices.
  const isIVRMode = callData ? (callData.leg2_total ?? 0) === 0 : true;

  const pieData = callData
    ? [
        { name: 'Connected',   value: callData.connected_calls,  color: CALL_CHART_COLORS.connected },
        { name: 'Leg 2 Drop',  value: callData.leg2_drops ?? 0,  color: CALL_CHART_COLORS.dropped },
        { name: 'Leg 1 Drop',  value: callData.leg1_drops ?? 0,  color: '#faad14' },
        { name: 'Missed',      value: callData.no_answer_calls,   color: CALL_CHART_COLORS.missed },
        { name: 'Failed',      value: callData.failed_calls,      color: CALL_CHART_COLORS.failed },
        { name: 'Busy',        value: callData.busy_calls,        color: CALL_CHART_COLORS.busy },
        { name: 'Canceled',    value: callData.canceled_calls,    color: CALL_CHART_COLORS.canceled },
        { name: 'Other',       value: callData.other_calls,       color: CALL_CHART_COLORS.other },
      ].filter((d) => d.value > 0)
    : [];

  const hourlyDist = parseHourlyDistribution(callData?.hourly_distribution);
  const hourlyChartData = hourlyDist.map((count, h) => ({
    hour: fmtHour(h),
    calls: count,
    isPeak: callData ? h === callData.peak_hour : false,
  }));

  // Staleness banner shown when today's data isn't ready yet
  const staleLabel = isStale && dataAsOf
    ? `Showing data from ${dayjs(dataAsOf).format('MMM D, YYYY')} — today's data will appear after the next sync`
    : null;

  // Card title with date picker
  const cardTitle = (
    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 8 }}>
      <Space>
        <span>Call Analytics</span>
        {isStale && (
          <Tag icon={<ClockCircleOutlined />} color="warning">
            Last data: {dataAsOf ? dayjs(dataAsOf).format('MMM D') : '—'}
          </Tag>
        )}
        {!isStale && !selectedDate && (
          <Tag color="default" style={{ fontWeight: 'normal', fontSize: 11 }}>Today</Tag>
        )}
        {selectedDate && (
          <Tag color="blue">{selectedDate.format('MMM D, YYYY')}</Tag>
        )}
      </Space>
      <DatePicker
        size="small"
        value={selectedDate}
        onChange={setSelectedDate}
        disabledDate={(d) => d.isAfter(dayjs())}
        allowClear
        placeholder="Pick date"
      />
    </div>
  );

  if (loadingMetrics) return <LoadingSpinner size="large" />;

  if (!metrics) {
    return (
      <EmptyState
        description="No health snapshot available for this exophone."
        onRetry={refetchMetrics}
      />
    );
  }

  return (
    <div>
      {/* Header */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <Space>
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/exophones')} type="text" />
          <div>
            <Title level={3} style={{ margin: 0 }}>{displayNumber}</Title>
            <Space size={8}>
              <Text type="secondary">{accountName}</Text>
              {exophone?.priority && (
                <Tag color={PRIORITY_COLOR[exophone.priority] ?? 'default'}>{exophone.priority}</Tag>
              )}
              <HeartbeatBadge status={metrics.heartbeat_status} />
            </Space>
          </div>
        </Space>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={handleRefresh} size="small">Refresh</Button>
          <Button icon={<DownloadOutlined />} onClick={handleExportCSV} size="small" disabled={!accountID}>
            Export CSV
          </Button>
          <Tooltip title={effectiveSkip ? 'Call logs are being skipped — toggle OFF to write all calls to DB' : 'Call logs are being written to DB — toggle ON to skip writing'}>
            <Space size={4}>
              <Switch
                size="small"
                checked={effectiveSkip}
                loading={savingSkip}
                onChange={handleToggleSkipCallLogs}
              />
              <Text style={{ fontSize: 11, color: effectiveSkip ? '#ff4d4f' : '#52c41a' }}>
                {effectiveSkip ? 'Skip call logs' : 'Write call logs'}
              </Text>
            </Space>
          </Tooltip>
          <Popconfirm
            title="Rebuild snapshot from raw API responses?"
            description={`Overwrites today's snapshot for ${displayNumber} with a fresh recomputation from saved job responses. Only works if job responses were saved.`}
            onConfirm={handleReprocess}
            okText="Reprocess"
            cancelText="Cancel"
          >
            <Button icon={<SyncOutlined spin={reprocessing} />} size="small" loading={reprocessing} danger>
              Reprocess
            </Button>
          </Popconfirm>
        </Space>
      </div>

      <Row gutter={[16, 16]}>
        {/* Left — KPIs + analytics */}
        <Col xs={24} lg={16}>
          <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
            <Col xs={8}>
              <KpiCard
                title="Availability"
                value={metrics.availability_percent.toFixed(1)}
                unit="%"
                highlight={metrics.availability_percent >= 99 ? 'success' : metrics.availability_percent >= 90 ? 'warning' : 'danger'}
              />
            </Col>
            <Col xs={8}>
              <KpiCard
                title="API Latency"
                value={metrics.last_api_latency_ms ?? '—'}
                unit={metrics.last_api_latency_ms != null ? 'ms' : undefined}
                highlight={
                  metrics.last_api_latency_ms == null ? undefined
                  : metrics.last_api_latency_ms < 1000 ? 'success'
                  : metrics.last_api_latency_ms < 3000 ? 'warning'
                  : 'danger'
                }
              />
            </Col>
            <Col xs={8}>
              <KpiCard title="Active Streams" value={metrics.active_streams} />
            </Col>
          </Row>

          <Card title={cardTitle} loading={loadingCalls}>
            {/* Stale data banner */}
            {staleLabel && (
              <Alert
                type="info"
                showIcon
                icon={<ClockCircleOutlined />}
                message={staleLabel}
                style={{ marginBottom: 16 }}
              />
            )}

            {/* Pending — no data ever collected for this exophone */}
            {isPending2 && (
              <Alert
                type="info"
                showIcon
                message="First sync pending"
                description="No call data has been collected for this exophone yet. Data will appear here automatically once the next monitoring job runs."
                style={{ marginBottom: 16 }}
              />
            )}

            {(!callData || callData.total_calls === 0) && !isPending2 ? (
              <EmptyState
                description={
                  isStale
                    ? `No calls recorded on ${dataAsOf ? dayjs(dataAsOf).format('MMM D, YYYY') : 'the last available date'}.`
                    : `No call data for ${selectedDate ? selectedDate.format('MMM D, YYYY') : 'today'}.`
                }
              />
            ) : callData ? (
              <>
                {/* Pie + stats row */}
                <Row gutter={[16, 0]}>
                  <Col xs={24} sm={12}>
                    <ResponsiveContainer width="100%" height={220}>
                      <PieChart>
                        <Pie
                          data={pieData}
                          dataKey="value"
                          nameKey="name"
                          cx="50%"
                          cy="50%"
                          outerRadius={80}
                          label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
                        >
                          {pieData.map((entry) => (
                            <Cell key={entry.name} fill={entry.color} />
                          ))}
                        </Pie>
                        <RechartsTooltip formatter={(v: number) => [v, 'calls']} />
                        <Legend />
                      </PieChart>
                    </ResponsiveContainer>
                  </Col>
                  <Col xs={24} sm={12}>
                    <Descriptions column={1} size="small" style={{ marginTop: 16 }}>
                      <Descriptions.Item label="Total Calls">{callData.total_calls}</Descriptions.Item>
                      <Descriptions.Item label="Connected">
                        <Text style={{ color: CALL_CHART_COLORS.connected }}>{callData.connected_calls}</Text>
                      </Descriptions.Item>
                      <Descriptions.Item label={<span>Leg 2 Drop <Text type="secondary" style={{ fontSize: 10 }}>(Leg2 not answered)</Text></span>}>
                        <Text style={{ color: (callData.leg2_drops ?? 0) > 0 ? CALL_CHART_COLORS.dropped : undefined }}>
                          {callData.leg2_drops ?? 0}
                          {(callData.leg2_drops ?? 0) > 0 && (
                            <Text type="danger" style={{ marginLeft: 4, fontSize: 11 }}>
                              ({fmtPercent(callData.drop_rate)})
                            </Text>
                          )}
                        </Text>
                      </Descriptions.Item>
                      <Descriptions.Item label={<span>Leg 1 Drop <Text type="secondary" style={{ fontSize: 10 }}>(dropped before Leg2)</Text></span>}>
                        <Text style={{ color: (callData.leg1_drops ?? 0) > 0 ? '#faad14' : undefined }}>
                          {callData.leg1_drops ?? 0}
                          {(callData.leg1_drops ?? 0) > 0 && (
                            <Text style={{ marginLeft: 4, fontSize: 11, color: '#faad14' }}>
                              ({fmtPercent(callData.leg1_drop_rate ?? 0)})
                            </Text>
                          )}
                        </Text>
                      </Descriptions.Item>
                      {callData.failed_calls > 0 && (
                        <Descriptions.Item label="Failed">
                          <Text type="danger">{callData.failed_calls}</Text>
                        </Descriptions.Item>
                      )}
                      {callData.no_answer_calls > 0 && (
                        <Descriptions.Item label="Missed (No Answer)">
                          <Text style={{ color: CALL_CHART_COLORS.missed }}>{callData.no_answer_calls}</Text>
                        </Descriptions.Item>
                      )}
                      {callData.busy_calls > 0 && (
                        <Descriptions.Item label="Busy">
                          <Text style={{ color: CALL_CHART_COLORS.busy }}>{callData.busy_calls}</Text>
                        </Descriptions.Item>
                      )}
                      {callData.canceled_calls > 0 && (
                        <Descriptions.Item label="Canceled">
                          <Text style={{ color: CALL_CHART_COLORS.canceled }}>{callData.canceled_calls}</Text>
                        </Descriptions.Item>
                      )}
                      {callData.other_calls > 0 && (
                        <Descriptions.Item label="Other">
                          <Text style={{ color: CALL_CHART_COLORS.other }}>{callData.other_calls}</Text>
                        </Descriptions.Item>
                      )}
                      <Descriptions.Item label="Answer Rate">
                        <Text style={{ color: callData.answer_rate >= 90 ? '#52c41a' : '#ff4d4f' }}>
                          {fmtPercent(callData.answer_rate)}
                        </Text>
                      </Descriptions.Item>
                      <Descriptions.Item label="Avg Duration">{fmtDuration(callData.avg_duration_sec)}</Descriptions.Item>
                      <Descriptions.Item label="Peak Hour">
                        <Tag>{fmtHour(callData.peak_hour)}</Tag>
                      </Descriptions.Item>
                    </Descriptions>

                    {/* Drop breakdown — always show when any drops exist */}
                    {((callData.leg2_drops ?? 0) > 0 || (callData.leg1_drops ?? 0) > 0) && (
                      <div style={{ marginTop: 12, padding: '10px 12px', background: '#fafafa', borderRadius: 6, border: '1px solid #f0f0f0' }}>
                        <Text strong style={{ fontSize: 12, display: 'block', marginBottom: 6 }}>
                          Drop Breakdown {isIVRMode && <Tag color="blue" style={{ fontSize: 10 }}>IVR Mode</Tag>}
                        </Text>
                        {(callData.leg2_drops ?? 0) > 0 && (
                          <div style={{ marginBottom: 4 }}>
                            <Text style={{ fontSize: 11, color: '#ff4d4f', fontWeight: 600 }}>Leg 2 Drop</Text>
                            <Text type="secondary" style={{ fontSize: 11 }}> — Leg2 was created but not answered · </Text>
                            <Text
                              style={{
                                fontSize: 11,
                                color: (callData.drop_rate ?? 0) >= 25 ? '#ff4d4f' : (callData.drop_rate ?? 0) >= 10 ? '#faad14' : '#52c41a',
                                fontWeight: 600,
                              }}
                            >
                              {fmtPercent(callData.drop_rate ?? 0)}
                            </Text>
                            <Text type="secondary" style={{ fontSize: 11 }}> ({callData.leg2_drops} / {callData.total_calls})</Text>
                          </div>
                        )}
                        {(callData.leg1_drops ?? 0) > 0 && (
                          <div>
                            <Text style={{ fontSize: 11, color: '#faad14', fontWeight: 600 }}>Leg 1 Drop</Text>
                            <Text type="secondary" style={{ fontSize: 11 }}> — dropped before Leg2 was created · </Text>
                            <Text
                              style={{
                                fontSize: 11,
                                color: (callData.leg1_drop_rate ?? 0) >= 25 ? '#ff4d4f' : (callData.leg1_drop_rate ?? 0) >= 10 ? '#faad14' : '#52c41a',
                                fontWeight: 600,
                              }}
                            >
                              {fmtPercent(callData.leg1_drop_rate ?? 0)}
                            </Text>
                            <Text type="secondary" style={{ fontSize: 11 }}> ({callData.leg1_drops} / {callData.total_calls})</Text>
                          </div>
                        )}
                      </div>
                    )}

                    {/* Leg status breakdown — from details=true Exotel API */}
                    {(() => {
                      const l1na = callData.leg1_no_answer ?? 0;
                      const l1busy = callData.leg1_busy ?? 0;
                      const l1fail = callData.leg1_failed ?? 0;
                      const l2na = callData.leg2_no_answer ?? 0;
                      const l2busy = callData.leg2_busy ?? 0;
                      const l2fail = callData.leg2_failed ?? 0;
                      const l2can = callData.leg2_canceled ?? 0;
                      const hasLegData = l1na + l1busy + l1fail + l2na + l2busy + l2fail + l2can > 0;
                      if (!hasLegData) return null;
                      const total = callData.total_calls || 1;
                      const pct = (n: number) => `${((n / total) * 100).toFixed(1)}%`;
                      const row = (label: string, val: number, color: string) =>
                        val > 0 ? (
                          <div key={label} style={{ display: 'flex', justifyContent: 'space-between', fontSize: 11, marginBottom: 2 }}>
                            <Text style={{ color }}>{label}</Text>
                            <Text type="secondary">{val} ({pct(val)})</Text>
                          </div>
                        ) : null;
                      return (
                        <div style={{ marginTop: 8, padding: '10px 12px', background: '#f6f6ff', borderRadius: 6, border: '1px solid #e8e8f0' }}>
                          <Text strong style={{ fontSize: 12, display: 'block', marginBottom: 6 }}>
                            Leg Status Reasons
                            <Text type="secondary" style={{ fontWeight: 'normal', marginLeft: 6, fontSize: 10 }}>from details=true API</Text>
                          </Text>
                          {(l1na > 0 || l1busy > 0 || l1fail > 0) && (
                            <div style={{ marginBottom: 6 }}>
                              <Text type="secondary" style={{ fontSize: 10, display: 'block', marginBottom: 2 }}>Leg 1</Text>
                              {row('No Answer', l1na, '#faad14')}
                              {row('Busy', l1busy, '#ff7a45')}
                              {row('Failed', l1fail, '#ff4d4f')}
                            </div>
                          )}
                          {(l2na > 0 || l2busy > 0 || l2fail > 0 || l2can > 0) && (
                            <div>
                              <Text type="secondary" style={{ fontSize: 10, display: 'block', marginBottom: 2 }}>Leg 2</Text>
                              {row('No Answer', l2na, '#faad14')}
                              {row('Busy', l2busy, '#ff7a45')}
                              {row('Failed', l2fail, '#ff4d4f')}
                              {row('Canceled', l2can, '#8c8c8c')}
                            </div>
                          )}
                        </div>
                      );
                    })()}
                  </Col>
                </Row>

                {/* Hourly distribution bar chart */}
                <div style={{ marginTop: 24 }}>
                  <Text strong style={{ display: 'block', marginBottom: 8 }}>
                    Hourly Distribution
                    <Tag color="blue" style={{ marginLeft: 8 }}>Peak: {fmtHour(callData.peak_hour)}</Tag>
                  </Text>
                  <ResponsiveContainer width="100%" height={180}>
                    <BarChart data={hourlyChartData} margin={{ top: 4, right: 8, left: -16, bottom: 0 }}>
                      <CartesianGrid strokeDasharray="3 3" vertical={false} />
                      <XAxis dataKey="hour" tick={{ fontSize: 10 }} interval={2} />
                      <YAxis tick={{ fontSize: 10 }} allowDecimals={false} />
                      <RechartsTooltip
                        formatter={(v: number) => [v, 'calls']}
                        labelFormatter={(l) => `Hour: ${l}`}
                      />
                      <Bar dataKey="calls" radius={[3, 3, 0, 0]}>
                        {hourlyChartData.map((entry, index) => (
                          <Cell
                            key={index}
                            fill={entry.isPeak ? '#1890ff' : '#52c41a'}
                            fillOpacity={entry.calls === 0 ? 0.2 : 0.85}
                          />
                        ))}
                      </Bar>
                    </BarChart>
                  </ResponsiveContainer>
                </div>
              </>
            ) : null}
          </Card>

          <Card title="Health Snapshot" style={{ marginTop: 16 }}>
            <Descriptions column={2} size="small">
              <Descriptions.Item label="Exophone">{displayNumber}</Descriptions.Item>
              <Descriptions.Item label="Organisation">{accountName}</Descriptions.Item>
              <Descriptions.Item label="Heartbeat"><HeartbeatBadge status={metrics.heartbeat_status} /></Descriptions.Item>
              <Descriptions.Item label="Availability">{fmtPercent(metrics.availability_percent)}</Descriptions.Item>
              <Descriptions.Item label="Last Latency">{fmtLatency(metrics.last_api_latency_ms)}</Descriptions.Item>
              <Descriptions.Item label="Active Streams">{metrics.active_streams}</Descriptions.Item>
              <Descriptions.Item label="Last Checked">{fmtRelative(metrics.last_checked_at)}</Descriptions.Item>
            </Descriptions>
          </Card>
        </Col>

        {/* Right — Alerts */}
        <Col xs={24} lg={8}>
          <Card
            title={
              <Space>
                Active Alerts
                {exophoneAlerts.length > 0 && (
                  <Tag color={exophoneAlerts.some((a) => a.severity === 'CRITICAL') ? 'red' : 'orange'}>
                    {exophoneAlerts.length}
                  </Tag>
                )}
              </Space>
            }
          >
            {exophoneAlerts.length === 0 ? (
              <Text type="secondary">No active alerts for {displayNumber}.</Text>
            ) : (
              <Space direction="vertical" style={{ width: '100%' }} size={8}>
                {exophoneAlerts.map((alert: AlertRecord) => (
                  <Card
                    key={alert.id}
                    size="small"
                    style={{
                      borderLeft: `3px solid ${alert.severity === 'CRITICAL' ? '#ff4d4f' : alert.severity === 'WARNING' ? '#faad14' : '#1890ff'}`,
                    }}
                  >
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                      <div style={{ flex: 1, marginRight: 8 }}>
                        <SeverityTag severity={alert.severity} />
                        <Tag style={{ marginLeft: 4 }}>{alert.alert_type.replace(/_/g, ' ')}</Tag>
                        <br />
                        <Text style={{ fontSize: 12 }}>{alert.message}</Text>
                        <br />
                        <Text type="secondary" style={{ fontSize: 11 }}>{fmtRelative(alert.triggered_at)}</Text>
                      </div>
                      <Popconfirm title="Acknowledge?" onConfirm={() => acknowledge(alert.id)} okText="Yes">
                        <Button size="small" loading={isPending}>Ack</Button>
                      </Popconfirm>
                    </div>
                  </Card>
                ))}
              </Space>
            )}
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default ExophoneDetail;
