import React, { Suspense } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { ConfigProvider, theme } from 'antd';
import AppShell from '@/components/layout/AppShell';
import LoadingSpinner from '@/components/common/LoadingSpinner';
import ErrorBoundary from '@/components/common/ErrorBoundary';

const Dashboard = React.lazy(() => import('@/pages/Dashboard'));
const ExophoneList = React.lazy(() => import('@/pages/Exophones'));
const ExophoneDetail = React.lazy(() => import('@/pages/ExophoneDetail'));
const Alerts = React.lazy(() => import('@/pages/Alerts'));
const Transactions = React.lazy(() => import('@/pages/Transactions'));
const SystemHealth = React.lazy(() => import('@/pages/SystemHealth'));

const App: React.FC = () => {
  return (
    <ConfigProvider
      theme={{
        algorithm: theme.defaultAlgorithm,
        token: {
          colorPrimary: '#1890ff',
          borderRadius: 6,
        },
      }}
    >
      <AppShell>
        {/* Each route is independently boundary-wrapped so one failing page
            never crashes the entire shell. */}
        <Suspense fallback={<LoadingSpinner size="large" />}>
          <Routes>
            <Route path="/" element={<Navigate to="/dashboard" replace />} />
            <Route path="/dashboard" element={<ErrorBoundary><Dashboard /></ErrorBoundary>} />
            <Route path="/exophones" element={<ErrorBoundary><ExophoneList /></ErrorBoundary>} />
            <Route path="/exophones/:id" element={<ErrorBoundary><ExophoneDetail /></ErrorBoundary>} />
            <Route path="/alerts" element={<ErrorBoundary><Alerts /></ErrorBoundary>} />
            <Route path="/transactions" element={<ErrorBoundary><Transactions /></ErrorBoundary>} />
            <Route path="/health" element={<ErrorBoundary><SystemHealth /></ErrorBoundary>} />
            <Route path="*" element={<Navigate to="/dashboard" replace />} />
          </Routes>
        </Suspense>
      </AppShell>
    </ConfigProvider>
  );
};

export default App;
