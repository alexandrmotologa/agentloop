import React, { useState } from 'react';
import { Execution } from '../types';
import { Layers, CheckCircle2, Clock, AlertTriangle, XCircle, ArrowRight } from 'lucide-react';

interface ExecutionsListProps {
  executions: Execution[];
  onSelectExecution: (id: string) => void;
  onOpenTrigger: () => void;
}

export const ExecutionsList: React.FC<ExecutionsListProps> = ({
  executions,
  onSelectExecution,
  onOpenTrigger,
}) => {
  const [filter, setFilter] = useState<string>('ALL');

  const filtered = executions.filter((e) => {
    if (filter === 'ALL') return true;
    return e.status === filter;
  });

  return (
    <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-6 pb-4 border-b border-slate-800">
        <div>
          <h2 className="text-lg font-bold text-slate-100 flex items-center gap-2">
            <Layers className="w-5 h-5 text-cyan-400" />
            Workflow Executions
          </h2>
          <p className="text-xs text-slate-400 mt-0.5">Historical and active workflow execution instances</p>
        </div>

        <div className="flex items-center gap-2">
          {/* Filters */}
          <div className="flex items-center gap-1 p-1 rounded-lg bg-slate-950 border border-slate-800 text-xs">
            {['ALL', 'RUNNING', 'COMPLETED', 'PAUSED_DLQ', 'FAILED'].map((st) => (
              <button
                key={st}
                onClick={() => setFilter(st)}
                className={`px-2.5 py-1 rounded font-medium transition-colors ${
                  filter === st
                    ? 'bg-cyan-500/20 text-cyan-300 border border-cyan-500/30'
                    : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                {st}
              </button>
            ))}
          </div>

          <button
            onClick={onOpenTrigger}
            className="px-3 py-1.5 rounded-lg text-xs font-semibold bg-cyan-500 text-slate-950 hover:bg-cyan-400 transition-colors"
          >
            Launch
          </button>
        </div>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-left text-xs">
          <thead>
            <tr className="border-b border-slate-800 text-slate-400 font-mono">
              <th className="pb-3 font-medium">EXECUTION ID</th>
              <th className="pb-3 font-medium">WORKFLOW SPEC</th>
              <th className="pb-3 font-medium">STATUS</th>
              <th className="pb-3 font-medium">LAST NODE</th>
              <th className="pb-3 font-medium">HEALING ATTEMPTS</th>
              <th className="pb-3 font-medium">STARTED AT</th>
              <th className="pb-3 text-right font-medium">ACTION</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-800/60 font-mono">
            {filtered.map((exec) => (
              <tr key={exec.id} className="hover:bg-slate-800/40 transition-colors group">
                <td className="py-3 text-cyan-400 font-bold">{exec.id}</td>
                <td className="py-3 text-slate-200 font-sans font-medium">{exec.workflow_id}</td>
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
                  {exec.status === 'FAILED' && (
                    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium bg-rose-500/10 text-rose-400 border border-rose-500/30">
                      <XCircle className="w-3 h-3" />
                      FAILED
                    </span>
                  )}
                </td>
                <td className="py-3 text-slate-400">{exec.current_node_id || '-'}</td>
                <td className="py-3 text-slate-400">{exec.healing_attempts}</td>
                <td className="py-3 text-slate-500 font-sans text-[11px]">
                  {new Date(exec.started_at).toLocaleString()}
                </td>
                <td className="py-3 text-right font-sans">
                  <button
                    onClick={() => onSelectExecution(exec.id)}
                    className="px-2.5 py-1 rounded text-xs font-medium bg-slate-800 text-slate-200 group-hover:bg-cyan-500 group-hover:text-slate-950 transition-colors flex items-center gap-1 ml-auto"
                  >
                    Details <ArrowRight className="w-3 h-3" />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};
