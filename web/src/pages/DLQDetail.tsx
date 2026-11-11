import React, { useState, useEffect } from 'react';
import { DLQItem, Execution } from '../types';
import { api } from '../api/client';
import { StateEditor } from '../components/StateEditor';
import { StateDiffView } from '../components/StateDiffView';
import { ArrowLeft, Play, RefreshCw, XOctagon, AlertOctagon, CheckCircle2, ShieldAlert, Sparkles, MessageSquare, SplitSquareVertical } from 'lucide-react';

interface DLQDetailProps {
  dlqId: string;
  onBack: () => void;
  onResolved: (executionId: string) => void;
}

export const DLQDetail: React.FC<DLQDetailProps> = ({ dlqId, onBack, onResolved }) => {
  const [item, setItem] = useState<DLQItem | null>(null);
  const [execution, setExecution] = useState<Execution | null>(null);
  const [initialState, setInitialState] = useState<Record<string, any>>({});
  const [frozenState, setFrozenState] = useState<Record<string, any>>({});
  const [activeTab, setActiveTab] = useState<'editor' | 'diff'>('editor');
  const [patch, setPatch] = useState<Record<string, any>>({});
  const [instruction, setInstruction] = useState('');
  const [loading, setLoading] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadData = async () => {
    try {
      const data = await api.getDLQItem(dlqId);
      setItem(data);

      try {
        const envelope = JSON.parse(data.snapshot_state_json);
        if (envelope && envelope.state) {
          setFrozenState(envelope.state);
        } else {
          setFrozenState(envelope);
        }
      } catch {
        setFrozenState({});
      }

      const execData = await api.getExecution(data.execution_id);
      setExecution(execData);

      // Load initial state from first snapshot for diff comparison
      try {
        const snaps = await api.getSnapshots(data.execution_id);
        if (snaps && snaps.length > 0) {
          const env = JSON.parse(snaps[0].state_json);
          setInitialState(env.state || env);
        } else if (execData) {
          setInitialState(execData.state || {});
        }
      } catch {
        // Fallback
      }
    } catch (err: any) {
      setError(err.message || 'Failed to load DLQ item');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, [dlqId]);

  const handleResume = async (retryNode: boolean) => {
    if (!item) return;
    setIsSubmitting(true);
    setError(null);
    try {
      const updatedExec = await api.resumeDLQ(item.id, {
        operator_instruction: instruction,
        patch: Object.keys(patch).length > 0 ? patch : undefined,
        retry_current_node: retryNode,
      });
      onResolved(updatedExec.id);
    } catch (err: any) {
      setError(err.message || 'Failed to resume execution');
      setIsSubmitting(false);
    }
  };

  const handleAbort = async () => {
    if (!item || !confirm('Are you sure you want to abort this execution permanently?')) return;
    setIsSubmitting(true);
    setError(null);
    try {
      await api.abortDLQ(item.id, instruction || 'Aborted by human operator');
      onResolved(item.execution_id);
    } catch (err: any) {
      setError(err.message || 'Failed to abort execution');
      setIsSubmitting(false);
    }
  };

  if (loading || !item) {
    return <div className="p-12 text-center text-xs text-slate-400">Loading DLQ incident state...</div>;
  }

  const isResolved = !!item.resolved_at;

  return (
    <div className="space-y-6">
      {/* Incident Header */}
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
                <span className="font-mono text-lg font-bold text-slate-100">{item.id}</span>
                {isResolved ? (
                  <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-emerald-500/10 text-emerald-400 border border-emerald-500/30">
                    <CheckCircle2 className="w-3.5 h-3.5" />
                    RESOLVED ({item.resolution_action})
                  </span>
                ) : (
                  <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-amber-500/10 text-amber-300 border border-amber-500/30 animate-pulse">
                    <AlertOctagon className="w-3.5 h-3.5" />
                    SUSPENDED (DLQ)
                  </span>
                )}
              </div>
              <p className="text-xs text-slate-400 mt-1">
                Execution: <span className="text-cyan-400 font-mono">{item.execution_id}</span>
                {execution && (
                  <span className="text-slate-400"> ({execution.workflow_id})</span>
                )}{' '}
                • Failed Node:{' '}
                <span className="text-amber-300 font-mono font-medium">{item.failed_node_id}</span>
              </p>
            </div>
          </div>

          <div className="text-xs text-slate-400 font-mono">
            <div>
              <span className="text-slate-500 block text-[10px]">CAPTURED AT</span>
              <span>{new Date(item.created_at).toLocaleString()}</span>
            </div>
          </div>
        </div>

        {/* Error Banner */}
        <div className="mt-5 p-4 rounded-xl bg-amber-950/20 border border-amber-900/50 flex flex-col gap-2">
          <div className="flex items-center gap-2 text-amber-400 font-semibold text-xs">
            <ShieldAlert className="w-4 h-4" />
            <span>Failure Reason: {item.failure_reason}</span>
          </div>
          {item.error_details && (
            <pre className="text-xs text-amber-200 font-mono whitespace-pre-wrap bg-slate-950/60 p-3 rounded-lg border border-amber-900/40">
              {item.error_details}
            </pre>
          )}
        </div>
      </div>

      {error && (
        <div className="p-4 rounded-xl bg-rose-500/10 border border-rose-500/30 text-rose-300 text-xs">
          {error}
        </div>
      )}

      {/* Main Grid: State Inspector & Operator Action Panel */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Left Column: Frozen State Editor / Visual Diff */}
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold text-slate-200 flex items-center gap-2">
              <Sparkles className="w-4 h-4 text-cyan-400" />
              State Inspection & Patching
            </h3>

            <div className="flex bg-slate-900 border border-slate-800 rounded-lg p-0.5 text-xs">
              <button
                onClick={() => setActiveTab('editor')}
                className={`px-3 py-1 rounded-md transition-colors ${
                  activeTab === 'editor'
                    ? 'bg-slate-800 text-cyan-300 font-medium'
                    : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                Patch Editor
              </button>
              <button
                onClick={() => setActiveTab('diff')}
                className={`flex items-center gap-1.5 px-3 py-1 rounded-md transition-colors ${
                  activeTab === 'diff'
                    ? 'bg-slate-800 text-cyan-300 font-medium'
                    : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                <SplitSquareVertical className="w-3.5 h-3.5" />
                Visual Diff
              </button>
            </div>
          </div>

          <p className="text-xs text-slate-400">
            {activeTab === 'editor'
              ? 'Inspect and edit the exact memory state at failure. Edits made below will be merged before resumption.'
              : 'Compare initial baseline state against the frozen state at failure to identify mutated variables.'}
          </p>

          {activeTab === 'editor' ? (
            <StateEditor
              initialState={frozenState}
              onSave={(updated) => setPatch(updated)}
              readOnly={isResolved}
            />
          ) : (
            <StateDiffView
              beforeState={initialState}
              afterState={frozenState}
              beforeTitle="Initial State"
              afterTitle={`Frozen Snapshot (${item.failed_node_id})`}
            />
          )}
        </div>

        {/* Right Column: Operator Steering & Actions */}
        <div className="space-y-6">
          <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-6 flex flex-col gap-5">
            <div>
              <h3 className="font-semibold text-slate-100 flex items-center gap-2 text-sm">
                <MessageSquare className="w-4 h-4 text-cyan-400" />
                Human-in-the-Loop Steering Directive
              </h3>
              <p className="text-xs text-slate-400 mt-1">
                Provide natural language guidance to the agentic node to unblock validation.
              </p>
            </div>

            <textarea
              value={instruction}
              onChange={(e) => setInstruction(e.target.value)}
              disabled={isResolved}
              placeholder="e.g. Vendor Tax ID was verified as valid with corporate registry. Proceed with approval."
              rows={5}
              className="w-full bg-slate-950 border border-slate-800 rounded-xl p-3 text-xs text-slate-200 focus:outline-none focus:border-cyan-500 disabled:opacity-50 resize-none font-sans leading-relaxed"
            />

            {item.operator_instruction && isResolved && (
              <div className="p-3 rounded-lg bg-slate-950 border border-slate-800 text-xs">
                <span className="text-slate-500 block text-[10px] mb-1">RECORDED OPERATOR DIRECTIVE:</span>
                <p className="text-slate-300 font-medium">{item.operator_instruction}</p>
              </div>
            )}

            {/* Action Buttons */}
            {!isResolved ? (
              <div className="pt-4 border-t border-slate-800/80 flex flex-col sm:flex-row gap-3">
                <button
                  onClick={() => handleResume(false)}
                  disabled={isSubmitting}
                  className="flex-1 py-2.5 px-4 rounded-xl text-xs font-semibold bg-gradient-to-r from-cyan-500 to-blue-600 hover:from-cyan-400 hover:to-blue-500 text-slate-950 flex items-center justify-center gap-2 transition-all shadow-md active:scale-95 disabled:opacity-50"
                >
                  <Play className="w-4 h-4 fill-slate-950" />
                  Resume & Advance Next
                </button>

                <button
                  onClick={() => handleResume(true)}
                  disabled={isSubmitting}
                  className="py-2.5 px-4 rounded-xl text-xs font-semibold bg-slate-800 hover:bg-slate-700 text-cyan-300 border border-slate-700 flex items-center justify-center gap-2 transition-colors disabled:opacity-50"
                >
                  <RefreshCw className="w-4 h-4" />
                  Retry Failed Node
                </button>

                <button
                  onClick={handleAbort}
                  disabled={isSubmitting}
                  className="py-2.5 px-3 rounded-xl text-xs font-semibold bg-rose-500/10 hover:bg-rose-500/20 text-rose-300 border border-rose-500/30 flex items-center justify-center gap-1.5 transition-colors disabled:opacity-50"
                >
                  <XOctagon className="w-4 h-4" />
                  Abort
                </button>
              </div>
            ) : (
              <div className="p-4 rounded-lg bg-emerald-500/10 border border-emerald-500/30 text-emerald-300 text-xs flex items-center gap-2 font-medium">
                <CheckCircle2 className="w-4 h-4 shrink-0" />
                This incident was resolved with action "{item.resolution_action}" by {item.resolved_by || 'operator'}.
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
