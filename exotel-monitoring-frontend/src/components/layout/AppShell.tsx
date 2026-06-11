import React, { useState, useEffect } from 'react';
import { Layout, Menu, Badge, Tag, Alert, Tooltip } from 'antd';
import {
  DashboardOutlined,
  PhoneOutlined,
  AlertOutlined,
  UnorderedListOutlined,
  HeartOutlined,
  WifiOutlined,
  DisconnectOutlined,
} from '@ant-design/icons';
import { useNavigate, useLocation } from 'react-router-dom';
import { useSystemHealth } from '@/hooks/useSystemHealth';
import { useActiveAlerts } from '@/hooks/useActiveAlerts';

const { Sider, Header, Content } = Layout;

const NAV_ITEMS = [
  { key: '/dashboard', icon: <DashboardOutlined />, label: 'Dashboard' },
  { key: '/exophones', icon: <PhoneOutlined />, label: 'Exophones' },
  { key: '/alerts', icon: <AlertOutlined />, label: 'Alerts' },
  { key: '/transactions', icon: <UnorderedListOutlined />, label: 'Transactions' },
  { key: '/health', icon: <HeartOutlined />, label: 'System Health' },
];

const AppShell: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [collapsed, setCollapsed] = useState(false);
  const [offline, setOffline] = useState(!navigator.onLine);
  const navigate = useNavigate();
  const location = useLocation();

  const { data: health } = useSystemHealth();
  const { data: alerts } = useActiveAlerts();

  useEffect(() => {
    const onOnline = () => setOffline(false);
    const onOffline = () => setOffline(true);
    window.addEventListener('online', onOnline);
    window.addEventListener('offline', onOffline);
    return () => {
      window.removeEventListener('online', onOnline);
      window.removeEventListener('offline', onOffline);
    };
  }, []);

  const isDegraded = health?.status === 'DEGRADED';
  const criticalCount = (alerts ?? []).filter((a) => a.severity === 'CRITICAL').length;
  const openCount = (alerts ?? []).length;

  const degradedServices: string[] = [];
  if (health?.mysql === 'DISCONNECTED') {
    degradedServices.push(`MySQL${health.mysql_error ? ': ' + health.mysql_error : ''}`);
  }
  if (health?.redis === 'DISCONNECTED') {
    degradedServices.push(`Redis${health.redis_error ? ': ' + health.redis_error : ''}`);
  }

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider
        collapsible
        collapsed={collapsed}
        onCollapse={setCollapsed}
        style={{ position: 'sticky', top: 0, height: '100vh', overflow: 'auto' }}
      >
        <div
          style={{
            padding: '16px',
            color: '#fff',
            fontWeight: 700,
            fontSize: collapsed ? 11 : 15,
            whiteSpace: 'nowrap',
            overflow: 'hidden',
            borderBottom: '1px solid rgba(255,255,255,0.1)',
            marginBottom: 8,
          }}
        >
          {collapsed ? 'EMP' : 'Exotel Monitor'}
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[location.pathname]}
          items={NAV_ITEMS.map((item) =>
            item.key === '/alerts' && openCount > 0
              ? {
                  ...item,
                  label: (
                    <span>
                      Alerts{' '}
                      <Badge
                        count={criticalCount || openCount}
                        color={criticalCount > 0 ? 'red' : 'orange'}
                        style={{ marginLeft: 4 }}
                      />
                    </span>
                  ),
                }
              : item
          )}
          onClick={({ key }) => navigate(key)}
        />
      </Sider>

      <Layout>
        <Header
          style={{
            background: '#fff',
            padding: '0 24px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            borderBottom: '1px solid #f0f0f0',
            position: 'sticky',
            top: 0,
            zIndex: 100,
          }}
        >
          <span style={{ fontWeight: 600, fontSize: 16 }}>Exotel Monitoring Platform</span>
          <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
            {offline && (
              <Tag icon={<DisconnectOutlined />} color="error">
                Offline
              </Tag>
            )}
            {health && (
              <Tooltip
                title={
                  isDegraded
                    ? degradedServices.join(' · ')
                    : 'All systems operational'
                }
              >
                <Tag
                  icon={isDegraded ? <DisconnectOutlined /> : <WifiOutlined />}
                  color={isDegraded ? 'error' : 'success'}
                  style={{ cursor: 'pointer' }}
                  onClick={() => navigate('/health')}
                >
                  System: {health.status}
                </Tag>
              </Tooltip>
            )}
            {openCount > 0 && (
              <Badge count={openCount} color={criticalCount > 0 ? 'red' : 'orange'}>
                <Tag
                  icon={<AlertOutlined />}
                  color={criticalCount > 0 ? 'error' : 'warning'}
                  style={{ cursor: 'pointer', margin: 0 }}
                  onClick={() => navigate('/alerts')}
                >
                  {criticalCount > 0 ? `${criticalCount} CRITICAL` : `${openCount} alerts`}
                </Tag>
              </Badge>
            )}
          </div>
        </Header>

        {/* Degradation banner shown on every page when services are down */}
        {isDegraded && (
          <Alert
            type="error"
            showIcon
            banner
            message={
              <span>
                <strong>Infrastructure Degraded —</strong>{' '}
                {degradedServices.join(' · ')}. Data shown may be stale or incomplete.{' '}
                <a onClick={() => navigate('/health')}>View details →</a>
              </span>
            }
          />
        )}

        {offline && (
          <Alert
            type="warning"
            showIcon
            banner
            message="You are offline. Data may be stale."
          />
        )}

        <Content style={{ padding: 24, minHeight: 'calc(100vh - 64px)' }}>
          {children}
        </Content>
      </Layout>
    </Layout>
  );
};

export default AppShell;
