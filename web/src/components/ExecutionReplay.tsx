import React, { useState } from 'react';
import { StepAuditEntry } from '../types';
import { CheckCircle2, AlertTriangle, XCircle, ChevronDown, ChevronRight, Clock, Zap } from 'lucide-react';

interface ExecutionReplayProps {
  entries: StepAuditEntry[];
}

export const ExecutionReplay: React.FC<ExecutionReplayProps> = ({ entries }) => {
  const [expandedId, setExpandedId] = useState<number | null>(null);

  const toggleExpand = (id: number) => {
    setExpandedId(expandedId === id ? null : id);
  };

  return (
    <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-5">
      <div className="flex items-center justify-between mb-4 pb-3 border-b border-slate-800">
        <div>
          <h3 className="font-semibold text-slate-100 flex items-center gap-2">
            <Zap className="w-4 h-4 text-cyan-400" />
            Step Audit Trail & Execution Replay
          </h3>
          <p className="text-xs text-slate-400 mt-0.5">
            Immutable compliance record with token telemetry and step payload captures
          </p>
        </div>
        <span className="text-xs font-mono text-slate-400">{entries.length} steps recorded</span>
      </div>

      {entries.length === 0 ? (
        <p className="text-xs text-slate-500 py-6 text-center">No step audit logs recorded yet.</p>
      ) : (
        <div className="space-y-2">
          {entries.map((entry, idx) => {
            const isExpanded = expandedId === entry.id;

            return (
              <div
                key={entry.id || idx}
                className="border border-slate-800/80 rounded-lg bg-slate-950/60 overflow-hidden"
              >
                <div
                  onClick={() => toggleExpand(entry.id)}
                  className="p-3 flex items-center justify-between cursor-pointer hover:bg-slate-800/40 transition-colors"
                >
                  <div className="flex items-center gap-3">
                    <span className="text-xs font-mono text-slate-500 w-5">#{idx + 1}</span>
                    <div className="flex items-center gap-2">
                      <span className="font-semibold text-xs text-slate-200">{entry.node_id}</span>
                      <span className="text-[10px] font-mono px-1.5 py-0.2 rounded bg-slate-800 text-slate-400">
                        {entry.node_type}
                      </span>
                    </div>
                  </div>

                  <div className="flex items-center gap-4 text-xs font-mono">
                    <span className="text-slate-400 flex items-center gap-1">
                      <Clock className="w-3 h-3 text-slate-500" />
                      {entry.latency_ms}ms
                    </span>
                    {entry.tokens_consumed > 0 && (
                      <span className="text-cyan-400">{entry.tokens_consumed} tokens</span>
                    )}

                    {entry.status === 'SUCCESS' && (
                      <span className="px-2 py-0.5 rounded text-[11px] font-medium bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 flex items-center gap-1">
                        <CheckCircle2 className="w-3 h-3" />
                        Success
                      </span>
                    )}
                    {entry.status === 'HEALED' && (
                      <span className="px-2 py-0.5 rounded text-[11px] font-medium bg-blue-500/10 text-blue-400 border border-blue-500/20 flex items-center gap-1">
                        <CheckCircle2 className="w-3 h-3" />
                        Healed
                      </span>
                    )}
                    {entry.status === 'PAUSED' && (
                      <span className="px-2 py-0.5 rounded text-[11px] font-medium bg-amber-500/10 text-amber-400 border border-amber-500/20 flex items-center gap-1">
                        <AlertTriangle className="w-3 h-3" />
                        DLQ Paused
                      </span>
                    )}
                    {entry.status === 'FAILED' && (
                      <span className="px-2 py-0.5 rounded text-[11px] font-medium bg-rose-500/10 text-rose-400 border border-rose-500/20 flex items-center gap-1">
                        <XCircle className="w-3 h-3" />
                        Failed
                      </span>
                    )}

                    {isExpanded ? (
                      <ChevronDown className="w-4 h-4 text-slate-400" />
                    ) : (
                      <ChevronRight className="w-4 h-4 text-slate-400" />
                    )}
                  </div>
                </div>

                {isExpanded && (
                  <div className="p-3.5 bg-slate-950 border-t border-slate-800/80 text-xs font-mono grid grid-cols-1 md:grid-cols-2 gap-3">
                    <div>
                      <span className="text-slate-400 font-semibold block mb-1">Input Payload:</span>
                      <pre className="p-2.5 rounded bg-slate-900 border border-slate-800 text-[11px] text-slate-300 overflow-x-auto max-h-40">
                        {entry.input_payload || 'null'}
                      </pre>
                    </div>
                    <div>
                      <span className="text-slate-400 font-semibold block mb-1">Output Payload:</span>
                      <pre className="p-2.5 rounded bg-slate-900 border border-slate-800 text-[11px] text-cyan-200 overflow-x-auto max-h-40">
                        {entry.output_payload || 'null'}
                      </pre>
                    </div>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
};
