import React, { useState } from 'react';
import { HealingLogEntry } from '../types';
import { Sparkles, AlertCircle, ChevronDown, ChevronRight, CheckCircle2, XCircle } from 'lucide-react';

interface HealingLogViewProps {
  entries: HealingLogEntry[];
}

export const HealingLogView: React.FC<HealingLogViewProps> = ({ entries }) => {
  const [expandedId, setExpandedId] = useState<number | null>(entries.length > 0 ? entries[0].id : null);

  const toggleExpand = (id: number) => {
    setExpandedId(expandedId === id ? null : id);
  };

  return (
    <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-5">
      <div className="flex items-center justify-between mb-4 pb-3 border-b border-slate-800">
        <div>
          <h3 className="font-semibold text-slate-100 flex items-center gap-2">
            <Sparkles className="w-4 h-4 text-cyan-400" />
            Autonomous Self-Healing Reflection Logs
          </h3>
          <p className="text-xs text-slate-400 mt-0.5">
            Diagnostic reflection sub-agent attempts, error classifications, and surgical prompt repairs
          </p>
        </div>
        <span className="text-xs font-mono text-cyan-400">{entries.length} repair attempts</span>
      </div>

      {entries.length === 0 ? (
        <p className="text-xs text-slate-500 py-6 text-center">
          No self-healing interventions triggered. Workflow executed on the first attempt without schema or tool faults.
        </p>
      ) : (
        <div className="space-y-3">
          {entries.map((entry) => {
            const isExpanded = expandedId === entry.id;

            return (
              <div
                key={entry.id}
                className="border border-slate-800 rounded-lg bg-slate-950/70 overflow-hidden"
              >
                <div
                  onClick={() => toggleExpand(entry.id)}
                  className="p-3.5 flex items-center justify-between cursor-pointer hover:bg-slate-800/40 transition-colors"
                >
                  <div className="flex items-center gap-3">
                    <span className="px-2 py-0.5 rounded text-xs font-bold font-mono bg-cyan-950 text-cyan-300 border border-cyan-800/60">
                      Attempt #{entry.attempt_number}
                    </span>
                    <span className="text-xs font-semibold text-slate-200">
                      Node: <span className="font-mono text-cyan-400">{entry.node_id}</span>
                    </span>
                    <span className="px-2 py-0.5 rounded text-[11px] font-mono bg-slate-800 text-amber-300 border border-amber-900/40">
                      {entry.error_category}
                    </span>
                  </div>

                  <div className="flex items-center gap-3">
                    {entry.status === 'SUCCESS' && (
                      <span className="px-2 py-0.5 rounded text-xs font-medium bg-emerald-500/10 text-emerald-400 border border-emerald-500/30 flex items-center gap-1">
                        <CheckCircle2 className="w-3.5 h-3.5" />
                        Repaired Successfully
                      </span>
                    )}
                    {entry.status === 'FAILED' && (
                      <span className="px-2 py-0.5 rounded text-xs font-medium bg-rose-500/10 text-rose-400 border border-rose-500/30 flex items-center gap-1">
                        <XCircle className="w-3.5 h-3.5" />
                        Retrying Next
                      </span>
                    )}
                    {entry.status === 'EXHAUSTED' && (
                      <span className="px-2 py-0.5 rounded text-xs font-medium bg-amber-500/10 text-amber-400 border border-amber-500/30 flex items-center gap-1">
                        <AlertCircle className="w-3.5 h-3.5" />
                        Exhausted & Frozen to DLQ
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
                  <div className="p-4 bg-slate-950 border-t border-slate-800/80 text-xs space-y-3">
                    <div>
                      <span className="text-slate-400 font-semibold block mb-1">Original Error Detected:</span>
                      <pre className="p-2.5 rounded bg-slate-900 border border-slate-800 font-mono text-rose-300 text-[11px] whitespace-pre-wrap">
                        {entry.original_error}
                      </pre>
                    </div>

                    <div>
                      <span className="text-slate-400 font-semibold block mb-1">
                        Synthesized Surgical Prompt Mutation:
                      </span>
                      <pre className="p-3 rounded bg-slate-900/90 border border-cyan-900/40 font-mono text-cyan-200 text-[11px] whitespace-pre-wrap max-h-60 overflow-y-auto leading-relaxed">
                        {entry.mutated_prompt}
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
