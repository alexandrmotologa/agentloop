import React, { useState, useEffect } from 'react';
import { Execution, WorkflowSpec, StepAuditEntry, HealingLogEntry } from '../types';
import { api } from '../api/client';
import { WorkflowDAGView } from '../components/WorkflowDAGView';
import { StateEditor } from '../components/StateEditor';
import { ExecutionReplay } from '../components/ExecutionReplay';
import { HealingLogView } from '../components/HealingLogView';
import {
  ArrowLeft,
  Clock,
  CheckCircle2,
  AlertTriangle,
  XCircle,
  FileCode,
  Layers,
  Sparkles,
  Download,
  Play,
  SkipForward,
  PauseCircle,
  Coins,
  FileText,
} from 'lucide-react';

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
  const [reportMarkdown, setReportMarkdown] = useState<string>('');
  const [activeTab, setActiveTab] = useState<'dag' | 'state' | 'audit' | 'healing' | 'report' | 'yaml'>('dag');
  const [loading, setLoading] = useState(true);
  const [debuggerLoading, setDebuggerLoading] = useState(false);

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
      setAuditLogs(Array.isArray(auditData) ? auditData : []);
      setHealingLogs(Array.isArray(healingData) ? healingData : []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const loadReport = async () => {
    try {
      const md = await api.getComplianceReport(executionId, 'markdown');
      setReportMarkdown(md);
    } catch (err) {
      console.error('Failed to load markdown report', err);
    }
  };

  useEffect(() => {
    loadData();
    const interval = setInterval(loadData, 3000);
    return () => clearInterval(interval);
  }, [executionId]);

  useEffect(() => {
    if (activeTab === 'report' && !reportMarkdown) {
      loadReport();
    }
  }, [activeTab]);

  const handleRewind = async (nodeId: string) => {
    if (!confirm(`Rewind execution state to checkpoint at node "${nodeId}"?`)) return;
    try {
      await api.rewindExecution(executionId, nodeId);
      loadData();
    } catch (err: any) {
      alert(err.message || 'Rewind failed');
    }
  };

  const handleToggleBreakpoint = async (nodeId: string) => {
    if (!execution) return;
    const currentBps = execution.breakpoints || [];
    const nextBps = currentBps.includes(nodeId)
      ? currentBps.filter((id) => id !== nodeId)
      : [...currentBps, nodeId];
    try {
      const updated = await api.setBreakpoints(executionId, nextBps);
      setExecution(updated);
    } catch (err: any) {
      alert(err.message || 'Failed to update breakpoint');
    }
  };

  const handleStep = async () => {
    setDebuggerLoading(true);
    try {
      const updated = await api.stepExecution(executionId);
      setExecution(updated);
      await loadData();
    } catch (err: any) {
      alert(err.message || 'Step execution failed');
    } finally {
      setDebuggerLoading(false);
    }
  };

  const handleContinue = async () => {
    setDebuggerLoading(true);
    try {
      const updated = await api.continueExecution(executionId);
      setExecution(updated);
      await loadData();
    } catch (err: any) {
      alert(err.message || 'Continue execution failed');
    } finally {
      setDebuggerLoading(false);
    }
  };

  const handleDownloadReport = async (format: 'markdown' | 'json') => {
    try {
      const data = await api.getComplianceReport(executionId, format);
      const blob = new Blob([typeof data === 'string' ? data : JSON.stringify(data, null, 2)], {
        type: format === 'markdown' ? 'text/markdown' : 'application/json',
      });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `compliance_report_${executionId}.${format === 'markdown' ? 'md' : 'json'}`;
      a.click();
      URL.revokeObjectURL(url);
    } catch (err: any) {
      alert(err.message || 'Failed to download compliance report');
    }
  };

  if (loading || !execution) {
    return <div className="p-12 text-center text-xs text-slate-400">Loading execution context...</div>;
  }

  const isHaltedBreakpoint = execution.status === 'HALTED_BREAKPOINT';

  return (
    <div className="space-y-6">
      {/* Breakpoint Active Sentinel Banner */}
      {isHaltedBreakpoint && (
        <div className="bg-amber-950/40 border border-amber-600/50 rounded-xl p-4 flex flex-col md:flex-row items-start md:items-center justify-between gap-4 animate-pulse">
          <div className="flex items-center gap-3">
            <PauseCircle className="w-5 h-5 text-amber-400 shrink-0" />
            <div>
              <h3 className="text-sm font-semibold text-amber-200">
                Execution Paused at Breakpoint: Node <code className="px-1.5 py-0.5 rounded bg-amber-900/60 font-mono font-bold text-amber-100">{execution.current_node_id}</code>
              </h3>
              <p className="text-xs text-amber-300/80 mt-0.5">
                The agent debugger intercepted execution before state mutation. Inspect memory state or advance step-by-step.
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={handleStep}
              disabled={debuggerLoading}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-amber-500 hover:bg-amber-400 text-slate-950 font-bold text-xs shadow-md transition-all active:scale-95 disabled:opacity-50 cursor-pointer"
            >
              <SkipForward className="w-4 h-4 fill-slate-950" />
              Step Next Node
            </button>
            <button
              onClick={handleContinue}
              disabled={debuggerLoading}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-bold text-xs shadow-md transition-all active:scale-95 disabled:opacity-50 cursor-pointer"
            >
              <Play className="w-4 h-4 fill-white" />
              Continue Run
            </button>
          </div>
        </div>
      )}

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
                {execution.status === 'HALTED_BREAKPOINT' && (
                  <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-amber-500/20 text-amber-300 border border-amber-500/40">
                    <PauseCircle className="w-3.5 h-3.5" />
                    HALTED_BREAKPOINT
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
                {spec?.workflow?.schedule && (
                  <span className="ml-2 px-2 py-0.5 rounded bg-indigo-950 text-indigo-300 border border-indigo-800/50">
                    Cron: {spec.workflow.schedule}
                  </span>
                )}
              </p>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-4 text-xs font-mono text-slate-400">
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
              <span className="text-slate-500 block text-[10px]">TOTAL TOKENS</span>
              <span className="text-slate-200 font-semibold">{execution.total_tokens || 0}</span>
            </div>
            <div>
              <span className="text-slate-500 block text-[10px]">EST. LLM COST</span>
              <span className="text-emerald-400 font-semibold flex items-center gap-0.5">
                <Coins className="w-3 h-3" />
                ${(execution.total_cost_usd || 0).toFixed(5)}
              </span>
            </div>
            <div>
              <span className="text-slate-500 block text-[10px]">HEALING ATTEMPTS</span>
              <span className="text-cyan-400 font-bold">{execution.healing_attempts}</span>
            </div>

            {/* Compliance Report Export Button */}
            <div className="flex items-center gap-1.5 pl-2 border-l border-slate-800">
              <button
                onClick={() => handleDownloadReport('markdown')}
                className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 font-sans text-xs transition-colors"
                title="Download Markdown Report"
              >
                <Download className="w-3.5 h-3.5 text-cyan-400" />
                Audit .md
              </button>
              <button
                onClick={() => handleDownloadReport('json')}
                className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 font-sans text-xs transition-colors"
                title="Download JSON Report"
              >
                <Download className="w-3.5 h-3.5 text-indigo-400" />
                Audit .json
              </button>
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
            Compliance Replay ({(auditLogs || []).length})
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
            Self-Healing Logs ({(healingLogs || []).length})
          </button>
          <button
            onClick={() => setActiveTab('report')}
            className={`px-3 py-1.5 rounded-lg font-medium transition-colors flex items-center gap-1.5 ${
              activeTab === 'report'
                ? 'bg-cyan-500/20 text-cyan-300 border border-cyan-500/30'
                : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/40'
            }`}
          >
            <FileText className="w-4 h-4" />
            Compliance Report
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
          onToggleBreakpoint={handleToggleBreakpoint}
        />
      )}

      {activeTab === 'state' && (
        <StateEditor initialState={execution.state} readOnly={execution.status === 'COMPLETED'} />
      )}

      {activeTab === 'audit' && <ExecutionReplay entries={auditLogs} />}

      {activeTab === 'healing' && <HealingLogView entries={healingLogs} />}

      {activeTab === 'report' && (
        <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-semibold text-slate-200 flex items-center gap-2">
              <FileText className="w-4 h-4 text-cyan-400" />
              Generated Compliance & Audit Report
            </h3>
            <div className="flex items-center gap-2">
              <button
                onClick={() => handleDownloadReport('markdown')}
                className="flex items-center gap-1.5 px-3 py-1 text-xs rounded bg-slate-800 hover:bg-slate-700 text-slate-300 transition-colors"
              >
                <Download className="w-3.5 h-3.5" />
                Export Markdown
              </button>
              <button
                onClick={() => handleDownloadReport('json')}
                className="flex items-center gap-1.5 px-3 py-1 text-xs rounded bg-slate-800 hover:bg-slate-700 text-slate-300 transition-colors"
              >
                <Download className="w-3.5 h-3.5" />
                Export JSON
              </button>
            </div>
          </div>
          <pre className="p-5 rounded-lg bg-slate-950 font-mono text-xs text-slate-300 overflow-x-auto whitespace-pre-wrap leading-relaxed border border-slate-800">
            {reportMarkdown || 'Loading compliance report...'}
          </pre>
        </div>
      )}

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
