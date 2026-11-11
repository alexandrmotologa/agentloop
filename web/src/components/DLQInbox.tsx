import React, { useState } from 'react';
import { DLQItem } from '../types';
import { AlertOctagon, CheckCircle2, Clock, ArrowRight, ShieldAlert } from 'lucide-react';

interface DLQInboxProps {
  items: DLQItem[];
  onSelect: (item: DLQItem) => void;
}

export const DLQInbox: React.FC<DLQInboxProps> = ({ items, onSelect }) => {
  const [filter, setFilter] = useState<'ALL' | 'PENDING' | 'RESOLVED'>('PENDING');

  const filteredItems = items.filter((item) => {
    if (filter === 'PENDING') return !item.resolved_at;
    if (filter === 'RESOLVED') return !!item.resolved_at;
    return true;
  });

  return (
    <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-6 pb-4 border-b border-slate-800">
        <div>
          <h3 className="font-semibold text-slate-100 flex items-center gap-2">
            <AlertOctagon className="w-5 h-5 text-amber-400" />
            Dead-Letter Queue (DLQ) Inbox
          </h3>
          <p className="text-xs text-slate-400 mt-0.5">
            Suspended workflows requiring operator intervention or audit confirmation
          </p>
        </div>

        {/* Status Filters */}
        <div className="flex items-center gap-1.5 p-1 rounded-lg bg-slate-950 border border-slate-800 self-start sm:self-auto">
          <button
            onClick={() => setFilter('PENDING')}
            className={`px-3 py-1 rounded-md text-xs font-medium transition-colors ${
              filter === 'PENDING'
                ? 'bg-amber-500/20 text-amber-300 border border-amber-500/30'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            Pending ({items.filter((i) => !i.resolved_at).length})
          </button>
          <button
            onClick={() => setFilter('RESOLVED')}
            className={`px-3 py-1 rounded-md text-xs font-medium transition-colors ${
              filter === 'RESOLVED'
                ? 'bg-slate-800 text-slate-200'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            Resolved ({items.filter((i) => !!i.resolved_at).length})
          </button>
          <button
            onClick={() => setFilter('ALL')}
            className={`px-3 py-1 rounded-md text-xs font-medium transition-colors ${
              filter === 'ALL'
                ? 'bg-slate-800 text-slate-200'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            All ({items.length})
          </button>
        </div>
      </div>

      {filteredItems.length === 0 ? (
        <div className="text-center py-12 border border-dashed border-slate-800 rounded-lg">
          <CheckCircle2 className="w-10 h-10 text-emerald-400/60 mx-auto mb-2" />
          <p className="text-slate-300 font-medium text-sm">No items matching current filter</p>
          <p className="text-slate-500 text-xs mt-1">Workflows are either running normally or already resolved</p>
        </div>
      ) : (
        <div className="space-y-3">
          {filteredItems.map((item) => {
            const isResolved = !!item.resolved_at;

            return (
              <div
                key={item.id}
                onClick={() => onSelect(item)}
                className={`p-4 rounded-xl border transition-all cursor-pointer flex flex-col md:flex-row md:items-center justify-between gap-4 group ${
                  isResolved
                    ? 'border-slate-800/80 bg-slate-900/40 hover:bg-slate-800/40'
                    : 'border-amber-900/50 bg-slate-900/80 hover:border-amber-700/60 hover:bg-slate-800/70 shadow-sm'
                }`}
              >
                <div className="flex items-start gap-3.5">
                  <div
                    className={`p-2.5 rounded-lg border mt-0.5 ${
                      isResolved
                        ? 'bg-slate-800 border-slate-700 text-slate-400'
                        : 'bg-amber-950/40 border-amber-800/60 text-amber-400'
                    }`}
                  >
                    {isResolved ? <CheckCircle2 className="w-5 h-5" /> : <ShieldAlert className="w-5 h-5" />}
                  </div>
                  <div>
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-mono text-xs font-bold text-slate-200">{item.id}</span>
                      <span className="text-slate-500 text-xs">•</span>
                      <span className="font-mono text-xs text-cyan-400">{item.execution_id}</span>
                      <span className="text-slate-500 text-xs">•</span>
                      <span className="text-xs font-medium text-slate-300 flex items-center gap-1">
                        Node: <span className="font-mono text-amber-300">{item.failed_node_id}</span>
                      </span>
                    </div>

                    <div className="mt-1.5 flex items-center gap-2">
                      <span
                        className={`inline-block px-2 py-0.5 rounded text-[11px] font-medium border ${
                          isResolved
                            ? 'bg-slate-800 text-slate-400 border-slate-700'
                            : 'bg-amber-500/10 text-amber-300 border-amber-500/30'
                        }`}
                      >
                        {item.failure_reason}
                      </span>
                      {item.error_details && (
                        <span className="text-xs text-slate-400 line-clamp-1 font-mono">
                          {item.error_details}
                        </span>
                      )}
                    </div>
                  </div>
                </div>

                <div className="flex items-center gap-4 self-end md:self-center">
                  <div className="text-right text-xs text-slate-500">
                    <div className="flex items-center gap-1 justify-end">
                      <Clock className="w-3.5 h-3.5" />
                      {new Date(item.created_at).toLocaleTimeString()}
                    </div>
                    {isResolved && (
                      <span className="text-emerald-400 text-[11px]">
                        Resolved ({item.resolution_action})
                      </span>
                    )}
                  </div>
                  <button className="px-3 py-1.5 rounded-lg text-xs font-medium bg-slate-800 text-slate-200 group-hover:bg-cyan-500 group-hover:text-slate-950 transition-colors flex items-center gap-1.5">
                    {isResolved ? 'Inspect' : 'Review & Resume'}
                    <ArrowRight className="w-3.5 h-3.5" />
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
};
