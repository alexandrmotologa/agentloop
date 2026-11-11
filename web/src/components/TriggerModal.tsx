import React, { useState } from 'react';
import { WorkflowSummary } from '../types';
import { X, Play, AlertCircle, FileText } from 'lucide-react';

interface TriggerModalProps {
  isOpen: boolean;
  onClose: () => void;
  workflows: WorkflowSummary[];
  onTrigger: (workflowId: string, input: Record<string, any>, asyncRun: boolean) => Promise<void>;
}

export const TriggerModal: React.FC<TriggerModalProps> = ({
  isOpen,
  onClose,
  workflows,
  onTrigger,
}) => {
  if (!isOpen) return null;

  const [selectedWorkflowId, setSelectedWorkflowId] = useState(
    workflows.length > 0 ? workflows[0].id : ''
  );
  const [inputJSON, setInputJSON] = useState('{\n  "invoice_pdf_path": "./inv_102.pdf"\n}');
  const [asyncRun, setAsyncRun] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedWorkflowId) {
      setError('Please select a workflow');
      return;
    }

    let parsedInput: Record<string, any> = {};
    try {
      parsedInput = JSON.parse(inputJSON);
    } catch (err: any) {
      setError('Invalid JSON: ' + err.message);
      return;
    }

    setIsSubmitting(true);
    setError(null);
    try {
      await onTrigger(selectedWorkflowId, parsedInput, asyncRun);
      onClose();
    } catch (err: any) {
      setError(err.message || 'Failed to trigger workflow');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/80 backdrop-blur-sm animate-in fade-in duration-200">
      <div className="w-full max-w-lg bg-slate-900 border border-slate-800 rounded-2xl shadow-2xl overflow-hidden flex flex-col">
        {/* Modal Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-800 bg-slate-950/60">
          <div className="flex items-center gap-2.5">
            <div className="p-2 rounded-lg bg-cyan-950 text-cyan-400 border border-cyan-800/50">
              <Play className="w-4 h-4 fill-cyan-400" />
            </div>
            <div>
              <h3 className="font-semibold text-slate-100">Launch Workflow Execution</h3>
              <p className="text-xs text-slate-400">Initialize execution state and trigger DAG engine</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 rounded-lg text-slate-400 hover:text-slate-200 hover:bg-slate-800 transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Modal Body */}
        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          {error && (
            <div className="p-3 rounded-lg bg-rose-500/10 border border-rose-500/30 text-rose-300 text-xs flex items-center gap-2">
              <AlertCircle className="w-4 h-4 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          <div>
            <label className="block text-xs font-semibold text-slate-300 mb-1.5">
              Select Workflow Specification
            </label>
            <select
              value={selectedWorkflowId}
              onChange={(e) => setSelectedWorkflowId(e.target.value)}
              className="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-xs text-slate-200 focus:outline-none focus:border-cyan-500 font-medium"
            >
              {workflows.map((wf) => (
                <option key={wf.id} value={wf.id}>
                  {wf.name} ({wf.id})
                </option>
              ))}
            </select>
          </div>

          <div>
            <div className="flex items-center justify-between mb-1.5">
              <label className="text-xs font-semibold text-slate-300 flex items-center gap-1.5">
                <FileText className="w-3.5 h-3.5 text-cyan-400" />
                Initial Input Payload (JSON)
              </label>
              <span className="text-[11px] text-slate-500 font-mono">merged into .state</span>
            </div>
            <textarea
              value={inputJSON}
              onChange={(e) => setInputJSON(e.target.value)}
              rows={6}
              className="w-full bg-slate-950 border border-slate-800 rounded-lg p-3 text-xs font-mono text-cyan-200 focus:outline-none focus:border-cyan-500 resize-none"
            />
          </div>

          <div className="flex items-center gap-2 pt-1">
            <input
              type="checkbox"
              id="asyncRun"
              checked={asyncRun}
              onChange={(e) => setAsyncRun(e.target.checked)}
              className="rounded bg-slate-950 border-slate-800 text-cyan-500 focus:ring-0"
            />
            <label htmlFor="asyncRun" className="text-xs text-slate-400 cursor-pointer">
              Execute asynchronously in the background (stream events via SSE)
            </label>
          </div>

          {/* Modal Actions */}
          <div className="flex items-center justify-end gap-3 pt-4 border-t border-slate-800">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 rounded-lg text-xs font-medium text-slate-400 hover:text-slate-200 hover:bg-slate-800 transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSubmitting}
              className="px-4 py-2 rounded-lg text-xs font-semibold bg-gradient-to-r from-cyan-500 to-blue-600 hover:from-cyan-400 hover:to-blue-500 text-slate-950 shadow-md shadow-cyan-500/10 flex items-center gap-1.5 transition-all disabled:opacity-50"
            >
              <Play className="w-3.5 h-3.5 fill-slate-950" />
              {isSubmitting ? 'Starting...' : 'Execute Workflow'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
