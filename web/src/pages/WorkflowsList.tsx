import React, { useState } from 'react';
import { WorkflowSummary } from '../types';
import { api } from '../api/client';
import { GitBranch, Play, FileCode, X } from 'lucide-react';

interface WorkflowsListProps {
  workflows: WorkflowSummary[];
  onOpenTriggerWithWorkflow: (id: string) => void;
}

export const WorkflowsList: React.FC<WorkflowsListProps> = ({
  workflows,
  onOpenTriggerWithWorkflow,
}) => {
  const [selectedWorkflowYAML, setSelectedWorkflowYAML] = useState<{ id: string; yaml: string } | null>(null);

  const viewSpec = async (id: string) => {
    try {
      const data = await api.getWorkflow(id);
      setSelectedWorkflowYAML({ id, yaml: data.raw_yaml });
    } catch (err: any) {
      alert(err.message || 'Failed to load spec');
    }
  };

  return (
    <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-6">
      <div className="flex items-center justify-between mb-6 pb-4 border-b border-slate-800">
        <div>
          <h2 className="text-lg font-bold text-slate-100 flex items-center gap-2">
            <GitBranch className="w-5 h-5 text-cyan-400" />
            Registered Workflow Specifications
          </h2>
          <p className="text-xs text-slate-400 mt-0.5">Declarative state machine blueprints</p>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {workflows.map((wf) => (
          <div
            key={wf.id}
            className="p-5 rounded-xl bg-slate-950/70 border border-slate-800/80 hover:border-slate-700 flex flex-col justify-between gap-4 group transition-all"
          >
            <div>
              <div className="flex items-start justify-between">
                <span className="font-mono text-xs font-bold text-cyan-400">{wf.id}</span>
                <span className="px-2 py-0.5 rounded text-[10px] font-mono bg-slate-800 text-slate-400 border border-slate-700">
                  {wf.node_count} nodes
                </span>
              </div>
              <h3 className="text-sm font-semibold text-slate-100 mt-2">{wf.name}</h3>
              <p className="text-xs text-slate-500 mt-1">
                Updated: {new Date(wf.updated_at).toLocaleDateString()}
              </p>
            </div>

            <div className="pt-3 border-t border-slate-800/80 flex items-center gap-2">
              <button
                onClick={() => viewSpec(wf.id)}
                className="flex-1 py-1.5 px-3 rounded-lg text-xs font-medium bg-slate-800 hover:bg-slate-700 text-slate-300 transition-colors flex items-center justify-center gap-1.5"
              >
                <FileCode className="w-3.5 h-3.5" />
                View Spec
              </button>
              <button
                onClick={() => onOpenTriggerWithWorkflow(wf.id)}
                className="py-1.5 px-3 rounded-lg text-xs font-semibold bg-cyan-500 hover:bg-cyan-400 text-slate-950 transition-colors flex items-center justify-center gap-1.5"
              >
                <Play className="w-3.5 h-3.5 fill-slate-950" />
                Run
              </button>
            </div>
          </div>
        ))}
      </div>

      {/* YAML Viewer Modal */}
      {selectedWorkflowYAML && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/80 backdrop-blur-sm animate-in fade-in">
          <div className="w-full max-w-2xl bg-slate-900 border border-slate-800 rounded-2xl shadow-2xl overflow-hidden flex flex-col">
            <div className="flex items-center justify-between px-6 py-4 border-b border-slate-800 bg-slate-950/60">
              <span className="font-mono text-xs font-bold text-cyan-400">
                {selectedWorkflowYAML.id} (YAML Spec)
              </span>
              <button
                onClick={() => setSelectedWorkflowYAML(null)}
                className="p-1 rounded text-slate-400 hover:text-slate-200"
              >
                <X className="w-5 h-5" />
              </button>
            </div>
            <pre className="p-6 font-mono text-xs text-cyan-200 overflow-y-auto max-h-[500px] leading-relaxed">
              {selectedWorkflowYAML.yaml}
            </pre>
          </div>
        </div>
      )}
    </div>
  );
};
