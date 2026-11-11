import React, { useState, useEffect } from 'react';
import { api, subscribeToEvents } from './api/client';
import { WorkflowSummary, Execution, DLQItem, ExecutionEvent } from './types';
import { Header } from './components/Header';
import { Dashboard } from './pages/Dashboard';
import { ExecutionsList } from './pages/ExecutionsList';
import { WorkflowDetail } from './pages/WorkflowDetail';
import { DLQDetail } from './pages/DLQDetail';
import { WorkflowsList } from './pages/WorkflowsList';
import { TriggerModal } from './components/TriggerModal';

export const App: React.FC = () => {
  const [activeTab, setActiveTab] = useState<string>('dashboard');
  const [selectedExecutionId, setSelectedExecutionId] = useState<string | null>(null);
  const [selectedDLQId, setSelectedDLQId] = useState<string | null>(null);

  const [workflows, setWorkflows] = useState<WorkflowSummary[]>([]);
  const [executions, setExecutions] = useState<Execution[]>([]);
  const [dlqItems, setDLQItems] = useState<DLQItem[]>([]);
  const [events, setEvents] = useState<ExecutionEvent[]>([]);
  const [isTriggerOpen, setIsTriggerOpen] = useState(false);

  const refreshData = async () => {
    try {
      const [wfs, execs, dlqs] = await Promise.all([
        api.listWorkflows(),
        api.listExecutions(),
        api.listDLQItems(),
      ]);
      setWorkflows(wfs);
      setExecutions(execs);
      setDLQItems(dlqs);
    } catch (err) {
      console.error('Error fetching dashboard data:', err);
    }
  };

  useEffect(() => {
    refreshData();
    const interval = setInterval(refreshData, 5000);
    return () => clearInterval(interval);
  }, []);

  // Real-time SSE streaming connection
  useEffect(() => {
    const unsubscribe = subscribeToEvents((event: ExecutionEvent) => {
      setEvents((prev) => [event, ...prev.slice(0, 99)]);
      // When execution state changes, refresh records
      refreshData();
    });

    return () => {
      unsubscribe();
    };
  }, []);

  const handleSelectExecution = (id: string) => {
    setSelectedExecutionId(id);
    setActiveTab('detail');
  };

  const handleSelectDLQ = (item: DLQItem) => {
    setSelectedDLQId(item.id);
    setActiveTab('dlq-detail');
  };

  const handleTrigger = async (workflowId: string, input: Record<string, any>, asyncRun: boolean) => {
    const exec = await api.runWorkflow(workflowId, input, asyncRun);
    await refreshData();
    if (!asyncRun) {
      handleSelectExecution(exec.id);
    }
  };

  const pendingDLQCount = dlqItems.filter((i) => !i.resolved_at).length;

  return (
    <div className="min-h-screen bg-[#070b14] flex flex-col font-['Plus_Jakarta_Sans',sans-serif] selection:bg-cyan-500/20 selection:text-cyan-300">
      <Header
        activeTab={activeTab}
        setActiveTab={(tab) => {
          setActiveTab(tab);
          if (tab !== 'detail') setSelectedExecutionId(null);
          if (tab !== 'dlq-detail') setSelectedDLQId(null);
        }}
        pendingDLQCount={pendingDLQCount}
        onOpenTrigger={() => setIsTriggerOpen(true)}
        onRefresh={refreshData}
      />

      <main className="flex-1 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 w-full">
        {activeTab === 'dashboard' && (
          <Dashboard
            workflows={workflows}
            executions={executions}
            dlqItems={dlqItems}
            events={events}
            onSelectExecution={handleSelectExecution}
            onSelectDLQ={handleSelectDLQ}
            onOpenTrigger={() => setIsTriggerOpen(true)}
          />
        )}

        {activeTab === 'executions' && (
          <ExecutionsList
            executions={executions}
            onSelectExecution={handleSelectExecution}
            onOpenTrigger={() => setIsTriggerOpen(true)}
          />
        )}

        {activeTab === 'dlq' && (
          <div className="space-y-6">
            <h2 className="text-lg font-bold text-slate-100">Dead-Letter Queue Review</h2>
            <Dashboard
              workflows={workflows}
              executions={executions}
              dlqItems={dlqItems}
              events={events}
              onSelectExecution={handleSelectExecution}
              onSelectDLQ={handleSelectDLQ}
              onOpenTrigger={() => setIsTriggerOpen(true)}
            />
          </div>
        )}

        {activeTab === 'workflows' && (
          <WorkflowsList
            workflows={workflows}
            onOpenTriggerWithWorkflow={() => {
              setIsTriggerOpen(true);
            }}
          />
        )}

        {activeTab === 'detail' && selectedExecutionId && (
          <WorkflowDetail
            executionId={selectedExecutionId}
            onBack={() => setActiveTab('executions')}
            onNavigateToDLQ={(dlqId) => {
              setSelectedDLQId(dlqId);
              setActiveTab('dlq-detail');
            }}
          />
        )}

        {activeTab === 'dlq-detail' && selectedDLQId && (
          <DLQDetail
            dlqId={selectedDLQId}
            onBack={() => setActiveTab('dashboard')}
            onResolved={(execId) => {
              refreshData();
              handleSelectExecution(execId);
            }}
          />
        )}
      </main>

      <TriggerModal
        isOpen={isTriggerOpen}
        onClose={() => setIsTriggerOpen(false)}
        workflows={workflows}
        onTrigger={handleTrigger}
      />

      <footer className="border-t border-slate-800/80 bg-slate-950/60 py-4 text-center text-xs text-slate-500 font-mono">
        agentloop v1.0.0 • Deterministic Self-Healing Enterprise Workflow Engine with Dead-Letter Queues & HITL
      </footer>
    </div>
  );
};
