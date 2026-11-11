import React from 'react';
import { Execution, DLQItem, WorkflowSummary, ExecutionEvent } from '../types';
import { DLQInbox } from '../components/DLQInbox';
import { Layers, Activity, Inbox, Zap, ArrowRight, CheckCircle2, AlertTriangle, Clock, Radio } from 'lucide-react';

interface DashboardProps {
  workflows: WorkflowSummary[];
  executions: Execution[];
  dlqItems: DLQItem[];
  events: ExecutionEvent[];
  onSelectExecution: (id: string) => void;
  onSelectDLQ: (item: DLQItem) => void;
  onOpenTrigger: () => void;
}

export const Dashboard: React.FC<DashboardProps> = ({
  workflows,
  executions,
  dlqItems,
  events,
  onSelectExecution,
  onSelectDLQ,
  onOpenTrigger,
}) => {
  const pendingDLQ = dlqItems.filter((i) => !i.resolved_at);
  const completedExecs = executions.filter((e) => e.status === 'COMPLETED');
  const runningExecs = executions.filter((e) => e.status === 'RUNNING');
  const pausedExecs = executions.filter((e) => e.status === 'PAUSED_DLQ');

  const recoveryRate = dlqItems.length > 0 
    ? Math.round((dlqItems.filter((i) => i.resolved_at).length / dlqItems.length) * 100) 
    : 100;

  return (
    <div className="space-y-6">
      {/* KPI Cards Row */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Card 1: Workflows */}
        <div className="p-5 rounded-xl bg-slate-900/60 border border-slate-800 relative overflow-hidden group hover:border-slate-700 transition-colors">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-slate-400">Registered Specs</span>
            <div className="p-2 rounded-lg bg-cyan-950/60 border border-cyan-800/40 text-cyan-400">
              <Layers className="w-4 h-4" />
            </div>
          </div>
          <div className="mt-3">
            <span className="text-2xl font-bold text-slate-100">{workflows.length}</span>
            <span className="text-xs text-slate-500 ml-2">DAG Workflows</span>
          </div>
          <div className="mt-2 text-xs text-cyan-400/80 flex items-center gap-1 font-mono">
            <span>Deterministic state machines</span>
          </div>
        </div>

        {/* Card 2: Active Executions */}
        <div className="p-5 rounded-xl bg-slate-900/60 border border-slate-800 relative overflow-hidden group hover:border-slate-700 transition-colors">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-slate-400">Execution Volume</span>
            <div className="p-2 rounded-lg bg-emerald-950/60 border border-emerald-800/40 text-emerald-400">
              <Activity className="w-4 h-4" />
            </div>
          </div>
          <div className="mt-3">
            <span className="text-2xl font-bold text-slate-100">{executions.length}</span>
            <span className="text-xs text-slate-500 ml-2">Total Runs</span>
          </div>
          <div className="mt-2 text-xs text-slate-400 flex items-center gap-2 font-mono">
            <span className="text-emerald-400">{completedExecs.length} completed</span>
            <span>•</span>
            <span className="text-cyan-400">{runningExecs.length} running</span>
          </div>
        </div>

        {/* Card 3: DLQ Interventions */}
        <div className="p-5 rounded-xl bg-slate-900/60 border border-slate-800 relative overflow-hidden group hover:border-slate-700 transition-colors">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-slate-400">DLQ Checkpoints</span>
            <div className="p-2 rounded-lg bg-amber-950/60 border border-amber-800/40 text-amber-400">
              <Inbox className="w-4 h-4" />
            </div>
          </div>
          <div className="mt-3">
            <span className="text-2xl font-bold text-slate-100">{pendingDLQ.length}</span>
            <span className="text-xs text-slate-500 ml-2">Pending Review</span>
          </div>
          <div className="mt-2 text-xs text-amber-400/90 flex items-center gap-1 font-mono">
            <span>{pausedExecs.length} frozen checkpoints</span>
          </div>
        </div>

        {/* Card 4: Engine Performance */}
        <div className="p-5 rounded-xl bg-slate-900/60 border border-slate-800 relative overflow-hidden group hover:border-slate-700 transition-colors">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-slate-400">Engine Overhead</span>
            <div className="p-2 rounded-lg bg-purple-950/60 border border-purple-800/40 text-purple-400">
              <Zap className="w-4 h-4" />
            </div>
          </div>
          <div className="mt-3">
            <span className="text-2xl font-bold text-slate-100">&lt; 10ms</span>
            <span className="text-xs text-slate-500 ml-2">Per Node</span>
          </div>
          <div className="mt-2 text-xs text-purple-400/90 flex items-center gap-1 font-mono">
            <span>{recoveryRate}% HITL resolution rate</span>
          </div>
        </div>
      </div>

      {/* Main Grid: DLQ Inbox & Live Stream */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Left 2 Cols: DLQ Inbox */}
        <div className="lg:col-span-2 space-y-6">
          <DLQInbox items={dlqItems} onSelect={onSelectDLQ} />

          {/* Recent Executions Table */}
          <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-6">
            <div className="flex items-center justify-between mb-4 pb-3 border-b border-slate-800">
              <div>
                <h3 className="font-semibold text-slate-100">Recent Executions</h3>
                <p className="text-xs text-slate-400 mt-0.5">Execution instances with live status tracking</p>
              </div>
              <button
                onClick={onOpenTrigger}
                className="text-xs text-cyan-400 hover:text-cyan-300 font-medium flex items-center gap-1"
              >
                Launch New <ArrowRight className="w-3.5 h-3.5" />
              </button>
            </div>

            {executions.length === 0 ? (
              <p className="text-xs text-slate-500 text-center py-6">No executions recorded yet.</p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs">
                  <thead>
                    <tr className="border-b border-slate-800 text-slate-400 font-mono">
                      <th className="pb-2.5 font-medium">EXECUTION ID</th>
                      <th className="pb-2.5 font-medium">WORKFLOW</th>
                      <th className="pb-2.5 font-medium">STATUS</th>
                      <th className="pb-2.5 font-medium">CURRENT NODE</th>
                      <th className="pb-2.5 font-medium">STARTED</th>
                      <th className="pb-2.5 text-right font-medium">ACTION</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-800/60 font-mono">
                    {executions.slice(0, 6).map((exec) => (
                      <tr key={exec.id} className="hover:bg-slate-800/40 transition-colors group">
                        <td className="py-3 text-cyan-400 font-bold">{exec.id}</td>
                        <td className="py-3 text-slate-300 font-sans font-medium">{exec.workflow_id}</td>
                        <td className="py-3">
                          {exec.status === 'COMPLETED' && (
                            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium bg-emerald-500/10 text-emerald-400 border border-emerald-500/30">
                              <CheckCircle2 className="w-3 h-3" />
                              COMPLETED
                            </span>
                          )}
                          {exec.status === 'RUNNING' && (
                            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium bg-cyan-500/10 text-cyan-400 border border-cyan-500/30 animate-pulse">
                              <Clock className="w-3 h-3" />
                              RUNNING
                            </span>
                          )}
                          {exec.status === 'PAUSED_DLQ' && (
                            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium bg-amber-500/10 text-amber-300 border border-amber-500/30 animate-pulse">
                              <AlertTriangle className="w-3 h-3" />
                              PAUSED_DLQ
                            </span>
                          )}
                          {exec.status === 'HALTED_BREAKPOINT' && (
                            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium bg-amber-500/20 text-amber-300 border border-amber-500/40">
                              BREAKPOINT
                            </span>
                          )}
                          {exec.status === 'FAILED' && (
                            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium bg-rose-500/10 text-rose-400 border border-rose-500/30">
                              FAILED
                            </span>
                          )}
                        </td>
                        <td className="py-3 text-slate-400">{exec.current_node_id || '-'}</td>
                        <td className="py-3 text-slate-500 font-sans text-[11px]">
                          {new Date(exec.started_at).toLocaleTimeString()}
                        </td>
                        <td className="py-3 text-right font-sans">
                          <button
                            onClick={() => onSelectExecution(exec.id)}
                            className="px-2.5 py-1 rounded text-xs font-medium bg-slate-800 text-slate-200 hover:bg-cyan-500 hover:text-slate-950 transition-colors"
                          >
                            Inspect
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>

        {/* Right 1 Col: Real-time Live Event Stream (SSE) */}
        <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-5 flex flex-col h-[700px]">
          <div className="flex items-center justify-between pb-3 border-b border-slate-800 mb-3">
            <h3 className="font-semibold text-slate-100 flex items-center gap-2 text-sm">
              <Radio className="w-4 h-4 text-cyan-400 animate-pulse" />
              Live SSE Event Stream
            </h3>
            <span className="px-1.5 py-0.5 text-[10px] font-mono rounded bg-emerald-950 text-emerald-400 border border-emerald-800/40">
              LIVE
            </span>
          </div>

          <div className="flex-1 overflow-y-auto space-y-2 pr-1 font-mono text-[11px]">
            {events.length === 0 ? (
              <p className="text-slate-500 text-center py-12 font-sans text-xs">
                Waiting for engine execution events...
              </p>
            ) : (
              events.map((ev, i) => (
                <div
                  key={i}
                  className="p-2.5 rounded-lg bg-slate-950/80 border border-slate-800/80 flex flex-col gap-1 hover:border-slate-700 transition-colors"
                >
                  <div className="flex items-center justify-between">
                    <span className="text-cyan-400 font-bold">{ev.type}</span>
                    <span className="text-slate-500 text-[10px]">
                      {new Date(ev.timestamp).toLocaleTimeString()}
                    </span>
                  </div>
                  {ev.node_id && (
                    <span className="text-slate-400">
                      Node: <span className="text-slate-200">{ev.node_id}</span>
                    </span>
                  )}
                  {ev.message && <p className="text-slate-300 font-sans text-xs">{ev.message}</p>}
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
