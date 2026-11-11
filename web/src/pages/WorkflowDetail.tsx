import React, { useState, useEffect } from 'react';
import { Execution, WorkflowSpec, StepAuditEntry, HealingLogEntry } from '../types';
import { api } from '../api/client';
import { WorkflowDAGView } from '../components/WorkflowDAGView';
import { StateEditor } from '../components/StateEditor';
import { ExecutionReplay } from '../components/ExecutionReplay';
import { HealingLogView } from '../components/HealingLogView';
import { ArrowLeft, Clock, CheckCircle2, AlertTriangle, XCircle, FileCode, Layers, Sparkles } from 'lucide-react';

interface WorkflowDetailProps {
  executionId: string;
  onBack: () => void;
  onNavigateToDLQ?: (dlqId: string) => void;
}

export const WorkflowDetail: React.FC<WorkflowDetailProps> = ({
  executionId,
  onBack,
  onNavigateToDLQ,
}) => {
  const [execution, setExecution] = useState<Execution | null>(null);
  const [spec, setSpec] = useState<WorkflowSpec | null>(null);
  const [rawYAML, setRawYAML] = useState<string>('');
  const [auditLogs, setAuditLogs] = useState<StepAuditEntry[]>([]);
  const [healingLogs, setHealingLogs] = useState<HealingLogEntry[]>([]);
  const [activeTab, setActiveTab] = useState<'dag' | 'state' | 'audit' | 'healing' | 'yaml'>('dag');
  const [loading, setLoading] = useState(true);

  const loadData = async () => {
    try {
      const execData = await api.getExecution(executionId);
      setExecution(execData);

      const [wfData, auditData, healingData] = await Promise.all([
        api.getWorkflow(execData.workflow_id),
        api.getAuditLogs(executionId),
        api.getHealingLogs(executionId),
      ]);

      setSpec(wfData.spec);
      setRawYAML(wfData.raw_yaml);
      setAuditLogs(auditData);
      setHealingLogs(healingData);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
    const interval = setInterval(loadData, 3000);
    return () => clearInterval(interval);
  }, [executionId]);

  const handleRewind = async (nodeId: string) => {
    if (!confirm(`Rewind execution state to checkpoint at node "${nodeId}"?`)) return;
    try {
      await api.rewindExecution(executionId, nodeId);
      loadData();
    } catch (err: any) {
      alert(err.message || 'Rewind failed');
    }
  };

  if (loading || !execution) {
    return <div className="p-12 text-center text-xs text-slate-400">Loading execution context...</div>;
  }

  return (
    <div className="space-y-6">
      {/* Top Bar with Navigation and Status */}
      <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-6">
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <button
              onClick={onBack}
              className="p-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 transition-colors"
            >
              <ArrowLeft className="w-4 h-4" />
            </button>
            <div>
              <div className="flex items-center gap-3">
                <h2 className="text-lg font-bold text-slate-100 font-mono">{execution.id}</h2>
                {execution.status === 'COMPLETED' && (
                  <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-emerald-500/10 text-emerald-400 border border-emerald-500/30">
                    <CheckCircle2 className="w-3.5 h-3.5" />
                    COMPLETED
                  </span>
                )}
                {execution.status === 'RUNNING' && (
                  <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-cyan-500/10 text-cyan-400 border border-cyan-500/30 animate-pulse">
                    <Clock className="w-3.5 h-3.5" />
                    RUNNING
                  </span>
                )}
                {execution.status === 'PAUSED_DLQ' && (
                  <button
                    onClick={() => onNavigateToDLQ && onNavigateToDLQ(execution.id)}
                    className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-amber-500/10 hover:bg-amber-500/20 text-amber-300 border border-amber-500/30 animate-pulse transition-colors cursor-pointer"
                  >
                    <AlertTriangle className="w-3.5 h-3.5" />
                    PAUSED_DLQ (Review in DLQ)
                  </button>
                )}
                {execution.status === 'FAILED' && (
                  <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-rose-500/10 text-rose-400 border border-rose-500/30">
                    <XCircle className="w-3.5 h-3.5" />
                    FAILED
                  </span>
                )}
              </div>
              <p className="text-xs text-slate-400 mt-1">
                Workflow Spec: <span className="text-slate-200 font-medium">{execution.workflow_id}</span>
              </p>
            </div>
          </div>

          <div className="flex items-center gap-4 text-xs font-mono text-slate-400">
            <div>
              <span className="text-slate-500 block text-[10px]">STARTED</span>
              <span>{new Date(execution.started_at).toLocaleTimeString()}</span>
            </div>
            {execution.completed_at && (
              <div>
                <span className="text-slate-500 block text-[10px]">COMPLETED</span>
                <span>{new Date(execution.completed_at).toLocaleTimeString()}</span>
              </div>
            )}
            <div>
              <span className="text-slate-500 block text-[10px]">HEALING ATTEMPTS</span>
              <span className="text-cyan-400 font-bold">{execution.healing_attempts}</span>
            </div>
          </div>
        </div>

        {/* Tab Navigation */}
        <div className="flex items-center gap-2 mt-6 pt-4 border-t border-slate-800 text-xs">
          <button
            onClick={() => setActiveTab('dag')}
            className={`px-3 py-1.5 rounded-lg font-medium transition-colors flex items-center gap-1.5 ${
              activeTab === 'dag'
                ? 'bg-cyan-500/20 text-cyan-300 border border-cyan-500/30'
                : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/40'
            }`}
          >
            <Layers className="w-4 h-4" />
            Interactive DAG
          </button>
          <button
            onClick={() => setActiveTab('state')}
            className={`px-3 py-1.5 rounded-lg font-medium transition-colors flex items-center gap-1.5 ${
              activeTab === 'state'
                ? 'bg-cyan-500/20 text-cyan-300 border border-cyan-500/30'
                : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/40'
            }`}
          >
            Memory State
          </button>
          <button
            onClick={() => setActiveTab('audit')}
            className={`px-3 py-1.5 rounded-lg font-medium transition-colors flex items-center gap-1.5 ${
              activeTab === 'audit'
                ? 'bg-cyan-500/20 text-cyan-300 border border-cyan-500/30'
                : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/40'
            }`}
          >
            Compliance Replay ({auditLogs.length})
          </button>
          <button
            onClick={() => setActiveTab('healing')}
            className={`px-3 py-1.5 rounded-lg font-medium transition-colors flex items-center gap-1.5 ${
              activeTab === 'healing'
                ? 'bg-cyan-500/20 text-cyan-300 border border-cyan-500/30'
                : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/40'
            }`}
          >
            <Sparkles className="w-4 h-4" />
            Self-Healing Logs ({healingLogs.length})
          </button>
          <button
            onClick={() => setActiveTab('yaml')}
            className={`px-3 py-1.5 rounded-lg font-medium transition-colors flex items-center gap-1.5 ${
              activeTab === 'yaml'
                ? 'bg-cyan-500/20 text-cyan-300 border border-cyan-500/30'
                : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/40'
            }`}
          >
            <FileCode className="w-4 h-4" />
            Spec YAML
          </button>
        </div>
      </div>

      {/* Tab Panels */}
      {activeTab === 'dag' && spec && (
        <WorkflowDAGView
          nodes={spec.nodes}
          execution={execution}
          auditLogs={auditLogs}
          onRewind={handleRewind}
        />
      )}

      {activeTab === 'state' && (
        <StateEditor initialState={execution.state} readOnly={execution.status === 'COMPLETED'} />
      )}

      {activeTab === 'audit' && <ExecutionReplay entries={auditLogs} />}

      {activeTab === 'healing' && <HealingLogView entries={healingLogs} />}

      {activeTab === 'yaml' && (
        <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-5">
          <pre className="p-4 rounded-lg bg-slate-950 font-mono text-xs text-cyan-200 overflow-x-auto leading-relaxed">
            {rawYAML}
          </pre>
        </div>
      )}
    </div>
  );
};
