import React, { useState, useMemo, useEffect } from 'react';
import {
  Table, Select, Button, Space, Typography, Tag, Row, Col, Card, Tabs, Switch,
  Popover, InputNumber, Form, notification, Modal, Input,
} from 'antd';
import { DatePicker } from 'antd';
import type { Dayjs } from 'dayjs';
import { ReloadOutlined, EyeOutlined, PlusOutlined, SyncOutlined, DownloadOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useExophones, useExophoneHealthList, useTodayPerformance, useToggleCron, useUpdatePriority, useSyncExophones, useAddExophone } from '@/hooks/useExophoneHealth';
import { useDashboardSummary } from '@/hooks/useDashboardSummary';
import HeartbeatBadge from '@/components/cards/HeartbeatBadge';
import EmptyState from '@/components/common/EmptyState';
import NewOrgModal from '@/components/forms/NewOrgModal';
import { fmtRelative, fmtPercent, fmtLatency } from '@/utils/formatters';
import { PRIORITY_COLOR } from '@/utils/constants';
import type { Exophone } from '@/types/exophone';
import type { ExophoneHealthSnapshot, CallMetricsSnapshot } from '@/types/snapshot';

const { Title, Text } = Typography;
const PAGE_SIZE = 20;

const Exophones: React.FC = () => {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const [showNewOrg, setShowNewOrg] = useState(false);
  const [showAddExophone, setShowAddExophone] = useState(false);
  const [addForm] = Form.useForm();

  const { data: summary, refetch: refetchSummary } = useDashboardSummary();
  const accounts = summary?.data ?? [];

  const defaultAccountId = searchParams.get('account_id')
    ? Number(searchParams.get('account_id'))
    : 0;

  const [accountId, setAccountId] = useState<number>(defaultAccountId);

  // Auto-select first account once the summary loads (state initialises before data arrives).
  useEffect(() => {
    if (!accountId && accounts.length > 0) {
      setAccountId(accounts[0].account_id);
    }
  }, [accounts]);
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [priorityFilter, setPriorityFilter] = useState<string>('all');
  const [exophonePage, setExophonePage] = useState(1);
  const [perfPage, setPerfPage] = useState(1);
  const [perfDate, setPerfDate] = useState<string | undefined>(undefined);

  const selectedAccount = accounts.find((a) => a.account_id === accountId);

  const {
    data: exophonePaged,
    isLoading: loadingExophones,
    isError: exophoneError,
    refetch: refetchExophones,
  } = useExophones(accountId, exophonePage, PAGE_SIZE);

  const exophones = exophonePaged?.data ?? [];
  const exophoneTotal = exophonePaged?.total ?? 0;

  const {
    data: healthList = [],
    isLoading: loadingHealth,
    refetch: refetchHealth,
  } = useExophoneHealthList(accountId);

  const {
    data: perfPaged,
    isLoading: loadingPerf,
    refetch: refetchPerf,
  } = useTodayPerformance(accountId, perfPage, PAGE_SIZE, perfDate);

  const { mutate: toggleCronFlag, isPending: togglingCron } = useToggleCron(accountId);
  const { mutate: changePriority, isPending: updatingPriority } = useUpdatePriority(accountId);
  const { mutate: syncVNs, isPending: syncing } = useSyncExophones(accountId);
  const { mutate: addVN, isPending: addingVN } = useAddExophone(accountId);

  const handleAddExophone = (values: { number: string; priority: string }) => {
    addVN(values, {
      onSuccess: (result) => {
        notification.success({
          message: 'Exophone added',
          description: result.message,
          duration: 5,
        });
        addForm.resetFields();
        setShowAddExophone(false);
        refetchExophones();
      },
      onError: (err: any) => {
        const msg = err?.response?.data?.error ?? err.message;
        notification.error({ message: 'Failed to add exophone', description: msg });
      },
    });
  };

  const handleSync = () => {
    if (!accountId) return;
    syncVNs(undefined, {
      onSuccess: (result) => {
        notification.success({
          message: 'Sync complete',
          description: `${result.total_from_exotel} VNs found on Exotel · ${result.newly_added} newly added · ${result.already_tracked} already tracked`,
          duration: 6,
        });
        refetchExophones();
      },
      onError: (err) => {
        notification.error({ message: 'Sync failed', description: err.message });
      },
    });
  };

  const DEFAULT_FREQ: Record<string, number> = { P0: 15, P1: 30, P2: 60, P3: 60 };

  const PriorityEditor: React.FC<{ exophone: Exophone }> = ({ exophone }) => {
    const [open, setOpen] = useState(false);
    const [selPriority, setSelPriority] = useState(exophone.priority);
    const [freq, setFreq] = useState<number>(DEFAULT_FREQ[exophone.priority] ?? 60);

    const handleSave = () => {
      changePriority(
        { exophoneId: exophone.id, priority: selPriority, frequencyMinutes: freq },
        { onSuccess: () => setOpen(false) },
      );
    };

    return (
      <Popover
        open={open}
        onOpenChange={setOpen}
        trigger="click"
        title="Edit Priority & Frequency"
        content={
          <div style={{ width: 220 }}>
            <Form layout="vertical" size="small">
              <Form.Item label="Priority" style={{ marginBottom: 8 }}>
                <Select
                  value={selPriority}
                  onChange={(v) => { setSelPriority(v); setFreq(DEFAULT_FREQ[v]); }}
                  options={[
                    { value: 'P0', label: 'P0 — Critical (default 15 min)' },
                    { value: 'P1', label: 'P1 — High (default 30 min)' },
                    { value: 'P2', label: 'P2 — Medium (default 60 min)' },
                    { value: 'P3', label: 'P3 — Low (default 60 min)' },
                  ]}
                />
              </Form.Item>
              <Form.Item label="Check every (minutes)" style={{ marginBottom: 12 }}>
                <InputNumber
                  min={1}
                  max={1440}
                  value={freq}
                  onChange={(v) => setFreq(v ?? DEFAULT_FREQ[selPriority])}
                  style={{ width: '100%' }}
                />
              </Form.Item>
              <Button
                type="primary"
                size="small"
                loading={updatingPriority}
                onClick={handleSave}
                block
              >
                Save
              </Button>
            </Form>
          </div>
        }
      >
        <Tag
          color={PRIORITY_COLOR[exophone.priority] ?? 'default'}
          style={{ cursor: 'pointer' }}
        >
          {exophone.priority}
        </Tag>
      </Popover>
    );
  };

  const perfRows = perfPaged?.data ?? [];
  const perfTotal = perfPaged?.total ?? 0;

  const healthMap = useMemo(() => {
    const m: Record<number, ExophoneHealthSnapshot> = {};
    healthList.forEach((h) => { m[h.exophone_id] = h; });
    return m;
  }, [healthList]);

  const filtered = useMemo(() =>
    exophones.filter((e) => {
      if (statusFilter !== 'all' && e.status !== statusFilter) return false;
      if (priorityFilter !== 'all' && e.priority !== priorityFilter) return false;
      return true;
    }),
    [exophones, statusFilter, priorityFilter],
  );

  const handleRefresh = () => {
    refetchSummary();
    if (accountId > 0) {
      refetchExophones();
      refetchHealth();
      refetchPerf();
    }
  };

  const handleAccountChange = (id: number) => {
    setAccountId(id);
    setExophonePage(1);
    setPerfPage(1);
  };

  const exophoneColumns: ColumnsType<Exophone> = [
    {
      title: 'Exophone Number',
      dataIndex: 'exophone_number',
      key: 'exophone_number',
      render: (num: string, row) => (
        <a onClick={() => navigate(`/exophones/${row.id}`)}>{num}</a>
      ),
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      render: (s: string) => <Tag color={s === 'active' ? 'green' : 'default'}>{s}</Tag>,
    },
    {
      title: 'Priority',
      key: 'priority',
      render: (_: unknown, row: Exophone) => <PriorityEditor exophone={row} />,
    },
    {
      title: 'Heartbeat',
      key: 'heartbeat',
      render: (_, row) => {
        const h = healthMap[row.id];
        return h ? <HeartbeatBadge status={h.heartbeat_status} /> : <Text type="secondary">—</Text>;
      },
    },
    {
      title: 'Availability',
      key: 'availability',
      render: (_, row) => fmtPercent(healthMap[row.id]?.availability_percent),
    },
    {
      title: 'Last Latency',
      key: 'latency',
      render: (_, row) => fmtLatency(healthMap[row.id]?.last_api_latency_ms),
    },
    {
      title: 'Last Synced',
      dataIndex: 'last_synced_at',
      key: 'last_synced_at',
      render: (v: string | null) => fmtRelative(v),
    },
    {
      title: 'Monitoring',
      dataIndex: 'monitoring_flag',
      key: 'monitoring_flag',
      render: (v: number) => <Tag color={v ? 'green' : 'red'}>{v ? '✓' : '✗'}</Tag>,
    },
    {
      title: 'Cron Active',
      dataIndex: 'is_cron_applicable',
      key: 'is_cron_applicable',
      render: (v: number, row) => (
        <Switch
          checked={v === 1}
          loading={togglingCron}
          onChange={(checked) =>
            toggleCronFlag({ exophoneId: row.id, val: checked ? 1 : 0 })
          }
        />
      ),
    },
    {
      title: '',
      key: 'action',
      width: 60,
      render: (_, row) => (
        <Button size="small" icon={<EyeOutlined />} onClick={() => navigate(`/exophones/${row.id}`)} />
      ),
    },
  ];

  // Build exophone number lookup from the already-loaded exophones list.
  const exophoneNumberMap = useMemo(() => {
    const m: Record<number, string> = {};
    exophones.forEach((e) => { m[e.id] = e.exophone_number; });
    return m;
  }, [exophones]);

  const perfColumns: ColumnsType<CallMetricsSnapshot> = [
    {
      title: 'Exophone',
      dataIndex: 'exophone_id',
      key: 'exophone_id',
      render: (id: number) => (
        <a onClick={() => navigate(`/exophones/${id}`)}>
          {exophoneNumberMap[id] ?? `#${id}`}
        </a>
      ),
    },
    { title: 'Total Calls', dataIndex: 'total_calls', key: 'total_calls' },
    {
      title: 'Connected',
      dataIndex: 'connected_calls',
      key: 'connected_calls',
      render: (v: number) => <Tag color="green">{v}</Tag>,
    },
    {
      title: 'Failed',
      dataIndex: 'failed_calls',
      key: 'failed_calls',
      render: (v: number) => <Tag color="red">{v}</Tag>,
    },
    { title: 'Busy', dataIndex: 'busy_calls', key: 'busy_calls' },
    {
      title: 'Success Rate',
      dataIndex: 'success_rate',
      key: 'success_rate',
      render: (v: number) => fmtPercent(v),
    },
    {
      title: 'Avg Duration',
      dataIndex: 'avg_duration_sec',
      key: 'avg_duration_sec',
      render: (v: number) => v ? `${v.toFixed(1)}s` : '—',
    },
  ];

  if (exophoneError) {
    return (
      <EmptyState
        description="Failed to load exophones. MySQL may be unavailable."
        onRetry={handleRefresh}
      />
    );
  }

  const tabItems = [
    {
      key: 'exophones',
      label: `Exophones${exophoneTotal > 0 ? ` (${exophoneTotal})` : ''}`,
      children: (
        <>
          {/* Health card grid */}
          {healthList.length > 0 && (
            <Row gutter={[12, 12]} style={{ marginBottom: 24 }}>
              {exophones.slice(0, 8).map((e) => {
                const h = healthMap[e.id];
                const borderColor = h?.heartbeat_status === 'OK' ? '#52c41a'
                  : h?.heartbeat_status === 'DEGRADED' ? '#faad14'
                  : h?.heartbeat_status === 'OUTAGE' ? '#ff4d4f'
                  : '#d9d9d9';
                return (
                  <Col key={e.id} xs={24} sm={12} md={8} lg={6}>
                    <Card
                      size="small"
                      hoverable
                      onClick={() => navigate(`/exophones/${e.id}`)}
                      style={{ borderLeft: `3px solid ${borderColor}` }}
                    >
                      <Text strong style={{ fontSize: 12 }}>{e.exophone_number}</Text>
                      <br />
                      <Text type="secondary" style={{ fontSize: 11 }}>
                        {selectedAccount?.account_name ?? `Account ${e.account_id}`} · {e.priority}
                      </Text>
                      <br />
                      {h ? (
                        <>
                          <HeartbeatBadge status={h.heartbeat_status} />
                          <Text type="secondary" style={{ fontSize: 11, marginLeft: 8 }}>
                            Avail: {fmtPercent(h.availability_percent)} · Streams: {h.active_streams}
                          </Text>
                        </>
                      ) : (
                        <Text type="secondary" style={{ fontSize: 11 }}>No health data</Text>
                      )}
                    </Card>
                  </Col>
                );
              })}
            </Row>
          )}

          {!loadingExophones && filtered.length === 0 ? (
            <EmptyState
              description={
                accountId
                  ? `No exophones found for ${selectedAccount?.account_name ?? 'this organisation'}.`
                  : 'Select an organisation to view its exophones, or create a new one.'
              }
              onRetry={accountId ? handleRefresh : undefined}
            />
          ) : (
            <Table
              dataSource={filtered}
              columns={exophoneColumns}
              rowKey="id"
              loading={loadingExophones || loadingHealth}
              pagination={{
                current: exophonePage,
                pageSize: PAGE_SIZE,
                total: exophoneTotal,
                showTotal: (t) => `${t} exophones`,
                onChange: (p) => setExophonePage(p),
                showSizeChanger: false,
              }}
              size="middle"
            />
          )}
        </>
      ),
    },
    {
      key: 'today',
      label: `Performance${perfTotal > 0 ? ` (${perfTotal})` : ''}`,
      children: (
        <>
          <div style={{ marginBottom: 12, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <Space>
              <Text type="secondary">Date:</Text>
              <DatePicker
                format="YYYY-MM-DD"
                placeholder="Today (default)"
                allowClear
                disabledDate={(d: Dayjs) => d && d.isAfter(new Date())}
                onChange={(_, dateStr) => {
                  setPerfDate(typeof dateStr === 'string' && dateStr ? dateStr : undefined);
                  setPerfPage(1);
                }}
              />
              {perfPaged?.date && (
                <Text type="secondary" style={{ fontSize: 12 }}>
                  Showing: {perfPaged.date}
                </Text>
              )}
            </Space>
            <Button
              icon={<DownloadOutlined />}
              size="small"
              disabled={!accountId || perfRows.length === 0}
              onClick={() => {
                const base = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080';
                const date = perfDate ?? new Date().toISOString().slice(0, 10);
                window.open(`${base}/api/v1/accounts/${accountId}/performance/export?date=${date}`, '_blank');
              }}
            >
              Export CSV
            </Button>
          </div>
          {!loadingPerf && perfRows.length === 0 ? (
            <EmptyState
              description={
                accountId
                  ? "No call metrics collected today yet — the scheduler hasn't run a job for this account."
                  : 'Select an organisation to view today\'s performance.'
              }
              onRetry={accountId ? refetchPerf : undefined}
            />
          ) : (
            <Table
              dataSource={perfRows}
              columns={perfColumns}
              rowKey="exophone_id"
              loading={loadingPerf}
              pagination={{
                current: perfPage,
                pageSize: PAGE_SIZE,
                total: perfTotal,
                showTotal: (t) => `${t} exophones with data today`,
                onChange: (p) => setPerfPage(p),
                showSizeChanger: false,
              }}
              size="middle"
            />
          )}
        </>
      ),
    },
  ];

  return (
    <div>
      {/* Header row */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Title level={3} style={{ margin: 0 }}>Exophones</Title>
        <Space>
          <Button icon={<PlusOutlined />} type="primary" onClick={() => setShowNewOrg(true)}>
            New Organisation
          </Button>
          <Button
            icon={<PlusOutlined />}
            onClick={() => setShowAddExophone(true)}
            disabled={!accountId}
          >
            Add Exophone
          </Button>
          <Button
            icon={<SyncOutlined />}
            onClick={handleSync}
            loading={syncing}
            disabled={!accountId}
            title="Fetch all active VNs from Exotel and sync new ones into monitoring"
          >
            Sync from Exotel
          </Button>
          <Button icon={<ReloadOutlined />} onClick={handleRefresh}>Refresh</Button>
        </Space>
      </div>

      {/* Account + filters */}
      <Space wrap style={{ marginBottom: 16 }}>
        <Select
          value={accountId || undefined}
          placeholder="Select organisation"
          onChange={handleAccountChange}
          style={{ minWidth: 220 }}
          options={accounts.map((a) => ({ value: a.account_id, label: a.account_name }))}
        />
        <Select
          value={statusFilter}
          onChange={setStatusFilter}
          style={{ width: 140 }}
          options={[
            { value: 'all', label: 'All Statuses' },
            { value: 'active', label: 'Active' },
            { value: 'inactive', label: 'Inactive' },
          ]}
        />
        <Select
          value={priorityFilter}
          onChange={setPriorityFilter}
          style={{ width: 140 }}
          options={[
            { value: 'all', label: 'All Priorities' },
            { value: 'P0', label: 'P0 — Critical' },
            { value: 'P1', label: 'P1 — High' },
            { value: 'P2', label: 'P2 — Medium' },
            { value: 'P3', label: 'P3 — Low' },
          ]}
        />
      </Space>

      <Tabs items={tabItems} defaultActiveKey="exophones" />

      <NewOrgModal open={showNewOrg} onClose={() => setShowNewOrg(false)} />

      <Modal
        title="Add Exophone"
        open={showAddExophone}
        onCancel={() => { setShowAddExophone(false); addForm.resetFields(); }}
        footer={null}
        width={420}
      >
        <Form form={addForm} layout="vertical" onFinish={handleAddExophone} style={{ marginTop: 12 }}>
          <Form.Item
            label="Virtual Number"
            name="number"
            rules={[{ required: true, message: 'Enter the virtual number' }]}
            extra="Enter the number as stored in Exotel (e.g. 07948224931)"
          >
            <Input placeholder="07948224931" />
          </Form.Item>
          <Form.Item label="Priority" name="priority" initialValue="P1">
            <Select
              options={[
                { value: 'P0', label: 'P0 — Critical (15 min)' },
                { value: 'P1', label: 'P1 — High (30 min)' },
                { value: 'P2', label: 'P2 — Medium (60 min)' },
                { value: 'P3', label: 'P3 — Low (60 min)' },
              ]}
            />
          </Form.Item>
          <Form.Item style={{ marginBottom: 0 }}>
            <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
              <Button onClick={() => { setShowAddExophone(false); addForm.resetFields(); }}>
                Cancel
              </Button>
              <Button type="primary" htmlType="submit" loading={addingVN}>
                Add &amp; Start Monitoring
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default Exophones;
