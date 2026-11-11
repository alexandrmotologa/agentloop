import React, { useState } from 'react';
import { NodeSpec, Execution, StepAuditEntry, NodeType } from '../types';
import { Bot, Wrench, GitFork, UserCheck, Code2, ArrowDown, CheckCircle2, AlertTriangle, XCircle, Clock, PlayCircle, RotateCcw } from 'lucide-react';

interface WorkflowDAGViewProps {
  nodes: NodeSpec[];
  execution?: Execution | null;
  auditLogs?: StepAuditEntry[];
  onRewind?: (nodeId: string) => void;
}

export const WorkflowDAGView: React.FC<WorkflowDAGViewProps> = ({
  nodes,
  execution,
  auditLogs = [],
  onRewind,
}) => {
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);

  const getNodeIcon = (type: NodeType) => {
    switch (type) {
      case 'llm_agent':
        return <Bot className="w-4 h-4 text-cyan-400" />;
      case 'tool_execution':
        return <Wrench className="w-4 h-4 text-emerald-400" />;
      case 'conditional_router':
        return <GitFork className="w-4 h-4 text-purple-400" />;
      case 'human_checkpoint':
        return <UserCheck className="w-4 h-4 text-amber-400" />;
      case 'transform':
        return <Code2 className="w-4 h-4 text-blue-400" />;
    }
  };

  const getNodeAudit = (nodeId: string) => {
    return auditLogs.find((log) => log.node_id === nodeId);
  };

  const getNodeState = (nodeId: string) => {
    if (!execution) return 'IDLE';

    const audit = getNodeAudit(nodeId);
    if (audit) {
      if (audit.status === 'HEALED') return 'HEALED';
      if (audit.status === 'SUCCESS') return 'SUCCESS';
      if (audit.status === 'FAILED') return 'FAILED';
      if (audit.status === 'PAUSED') return 'PAUSED';
    }

    if (execution.current_node_id === nodeId) {
      if (execution.status === 'PAUSED_DLQ') return 'PAUSED';
      if (execution.status === 'RUNNING') return 'RUNNING';
      if (execution.status === 'FAILED') return 'FAILED';
    }

    return 'PENDING';
  };

  const selectedNode = nodes.find((n) => n.id === selectedNodeId);
  const selectedAudit = selectedNodeId ? getNodeAudit(selectedNodeId) : null;

  return (
    <div className="flex flex-col lg:flex-row gap-6">
      {/* DAG Flow Visualizer */}
      <div className="flex-1 bg-slate-900/60 border border-slate-800 rounded-xl p-6 relative overflow-hidden">
        <div className="flex items-center justify-between mb-6 pb-4 border-b border-slate-800/80">
          <div>
            <h3 className="font-semibold text-slate-100 flex items-center gap-2">
              <span className="w-2.5 h-2.5 rounded-full bg-cyan-400 inline-block animate-pulse"></span>
              Execution Directed Acyclic Graph (DAG)
            </h3>
            <p className="text-xs text-slate-400 mt-0.5">
              Deterministic sequence with autonomous sentinels and fallback checkpoints
            </p>
          </div>
          <div className="text-xs text-slate-400 flex items-center gap-4">
            <span className="flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full bg-emerald-400"></span> Success
            </span>
            <span className="flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full bg-cyan-400 animate-pulse"></span> Running
            </span>
            <span className="flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full bg-amber-400"></span> DLQ Frozen
            </span>
          </div>
        </div>

        {/* Nodes Timeline / Column */}
        <div className="max-w-xl mx-auto flex flex-col items-center gap-3">
          {nodes.map((node, index) => {
            const state = getNodeState(node.id);
            const audit = getNodeAudit(node.id);
            const isSelected = selectedNodeId === node.id;

            return (
              <React.Fragment key={node.id}>
                <div
                  onClick={() => setSelectedNodeId(node.id)}
                  className={`w-full p-4 rounded-xl border transition-all cursor-pointer relative group ${
                    isSelected
                      ? 'border-cyan-500 bg-slate-800/90 shadow-lg shadow-cyan-500/10 scale-[1.01]'
                      : 'border-slate-800 bg-slate-900/90 hover:border-slate-700 hover:bg-slate-800/50'
                  }`}
                >
                  <div className="flex items-start justify-between">
                    <div className="flex items-center gap-3">
                      <div className="p-2 rounded-lg bg-slate-800 border border-slate-700/80">
                        {getNodeIcon(node.type)}
                      </div>
                      <div>
                        <div className="flex items-center gap-2">
                          <span className="font-semibold text-sm text-slate-200">{node.id}</span>
                          <span className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-slate-800 text-slate-400 border border-slate-700/50">
                            {node.type}
                          </span>
                        </div>
                        <p className="text-xs text-slate-400 mt-1 line-clamp-1">{node.description || 'Execution node'}</p>
                      </div>
                    </div>

                    {/* Status Badge */}
                    <div className="flex items-center gap-2">
                      {state === 'SUCCESS' && (
                        <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-emerald-500/10 text-emerald-400 border border-emerald-500/30">
                          <CheckCircle2 className="w-3.5 h-3.5" />
                          {audit ? `${audit.latency_ms}ms` : 'Passed'}
                        </span>
                      )}
                      {state === 'HEALED' && (
                        <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-blue-500/10 text-blue-400 border border-blue-500/30">
                          <CheckCircle2 className="w-3.5 h-3.5" />
                          Healed
                        </span>
                      )}
                      {state === 'RUNNING' && (
                        <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-cyan-500/10 text-cyan-400 border border-cyan-500/30 animate-pulse">
                          <PlayCircle className="w-3.5 h-3.5 animate-spin" />
                          Running
                        </span>
                      )}
                      {state === 'PAUSED' && (
                        <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-amber-500/10 text-amber-400 border border-amber-500/30 animate-pulse">
                          <AlertTriangle className="w-3.5 h-3.5" />
                          DLQ Paused
                        </span>
                      )}
                      {state === 'FAILED' && (
                        <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-rose-500/10 text-rose-400 border border-rose-500/30">
                          <XCircle className="w-3.5 h-3.5" />
                          Failed
                        </span>
                      )}
                      {state === 'PENDING' && (
                        <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-slate-800 text-slate-400">
                          <Clock className="w-3 h-3" />
                          Queued
                        </span>
                      )}
                    </div>
                  </div>

                  {/* Node Metrics Footer */}
                  {audit && (
                    <div className="mt-3 pt-2.5 border-t border-slate-800/80 flex items-center justify-between text-[11px] text-slate-400 font-mono">
                      <span>Latency: {audit.latency_ms}ms</span>
                      {audit.tokens_consumed > 0 && <span>Tokens: {audit.tokens_consumed}</span>}
                    </div>
                  )}
                </div>

                {/* Arrow Connector */}
                {index < nodes.length - 1 && (
                  <div className="flex flex-col items-center my-0.5 text-slate-600">
                    <ArrowDown className="w-4 h-4 animate-bounce" />
                  </div>
                )}
              </React.Fragment>
            );
          })}
        </div>
      </div>

      {/* Node Inspector Drawer */}
      {selectedNode && (
        <div className="w-full lg:w-96 bg-slate-900/90 border border-slate-800 rounded-xl p-5 flex flex-col gap-4">
          <div className="flex items-center justify-between pb-3 border-b border-slate-800">
            <div>
              <h4 className="font-semibold text-slate-100 flex items-center gap-2">
                {getNodeIcon(selectedNode.type)}
                {selectedNode.id}
              </h4>
              <span className="text-xs text-slate-400">{selectedNode.type}</span>
            </div>
            {onRewind && execution && (
              <button
                onClick={() => onRewind(selectedNode.id)}
                title="Rewind execution state to this node snapshot"
                className="px-2.5 py-1 text-xs font-medium rounded-md bg-slate-800 hover:bg-slate-700 text-cyan-400 border border-slate-700 flex items-center gap-1.5 transition-colors"
              >
                <RotateCcw className="w-3.5 h-3.5" />
                Rewind
              </button>
            )}
          </div>

          <div className="text-xs space-y-3 overflow-y-auto max-h-[500px]">
            {selectedNode.description && (
              <div>
                <span className="text-slate-400 font-medium">Description:</span>
                <p className="text-slate-200 mt-0.5">{selectedNode.description}</p>
              </div>
            )}

            {selectedNode.prompt && (
              <div>
                <span className="text-slate-400 font-medium">Prompt Template:</span>
                <pre className="mt-1 p-2.5 rounded bg-slate-950 border border-slate-800 text-[11px] text-cyan-200 font-mono whitespace-pre-wrap">
                  {selectedNode.prompt}
                </pre>
              </div>
            )}

            {selectedNode.tool && (
              <div>
                <span className="text-slate-400 font-medium">Tool Call:</span>
                <div className="mt-1 p-2 rounded bg-slate-950 border border-slate-800 font-mono text-emerald-400">
                  {selectedNode.tool}
                </div>
              </div>
            )}

            {selectedNode.output_schema && (
              <div>
                <span className="text-slate-400 font-medium">JSON Output Schema:</span>
                <pre className="mt-1 p-2.5 rounded bg-slate-950 border border-slate-800 text-[10px] text-slate-300 font-mono overflow-x-auto max-h-40">
                  {JSON.stringify(selectedNode.output_schema, null, 2)}
                </pre>
              </div>
            )}

            {selectedAudit && (
              <div>
                <span className="text-slate-400 font-medium">Recorded Execution Output:</span>
                <pre className="mt-1 p-2.5 rounded bg-slate-950 border border-slate-800 text-[10px] text-slate-200 font-mono overflow-x-auto max-h-44">
                  {selectedAudit.output_payload || 'No output payload recorded'}
                </pre>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
};
