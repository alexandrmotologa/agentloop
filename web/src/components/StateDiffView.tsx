import React, { useState } from 'react';
import { Copy, Check, Filter } from 'lucide-react';

interface StateDiffViewProps {
  beforeState: Record<string, any>;
  afterState: Record<string, any>;
  beforeTitle?: string;
  afterTitle?: string;
}

export const StateDiffView: React.FC<StateDiffViewProps> = ({
  beforeState = {},
  afterState = {},
  beforeTitle = 'Initial / Previous State',
  afterTitle = 'Frozen DLQ / Current State',
}) => {
  const [onlyChanges, setOnlyChanges] = useState(false);
  const [copiedBefore, setCopiedBefore] = useState(false);
  const [copiedAfter, setCopiedAfter] = useState(false);

  const allKeys = Array.from(new Set([...Object.keys(beforeState), ...Object.keys(afterState)])).sort();

  const diffItems = allKeys.map((key) => {
    const hasBefore = key in beforeState;
    const hasAfter = key in afterState;
    const valBefore = beforeState[key];
    const valAfter = afterState[key];

    const jsonBefore = hasBefore ? JSON.stringify(valBefore, null, 2) : undefined;
    const jsonAfter = hasAfter ? JSON.stringify(valAfter, null, 2) : undefined;

    let status: 'unchanged' | 'modified' | 'added' | 'removed' = 'unchanged';
    if (!hasBefore && hasAfter) {
      status = 'added';
    } else if (hasBefore && !hasAfter) {
      status = 'removed';
    } else if (jsonBefore !== jsonAfter) {
      status = 'modified';
    }

    return {
      key,
      status,
      valBefore,
      valAfter,
      jsonBefore,
      jsonAfter,
    };
  });

  const displayedItems = onlyChanges ? diffItems.filter((d) => d.status !== 'unchanged') : diffItems;

  const copyToClipboard = (text: string, isBefore: boolean) => {
    navigator.clipboard.writeText(text);
    if (isBefore) {
      setCopiedBefore(true);
      setTimeout(() => setCopiedBefore(false), 2000);
    } else {
      setCopiedAfter(true);
      setTimeout(() => setCopiedAfter(false), 2000);
    }
  };

  const changesCount = diffItems.filter((d) => d.status !== 'unchanged').length;

  return (
    <div className="flex flex-col gap-3 w-full">
      {/* Controls Bar */}
      <div className="flex items-center justify-between px-3 py-2 bg-slate-900 border border-slate-800 rounded-lg">
        <div className="flex items-center gap-2">
          <span className="text-xs font-semibold text-slate-300 uppercase tracking-wider">
            Visual State Differential
          </span>
          <span className="px-2 py-0.5 text-xs rounded-full bg-indigo-950 text-indigo-400 border border-indigo-800/50">
            {changesCount} {changesCount === 1 ? 'delta' : 'deltas'} detected
          </span>
        </div>

        <button
          onClick={() => setOnlyChanges(!onlyChanges)}
          className={`flex items-center gap-1.5 px-2.5 py-1 text-xs rounded transition-colors ${
            onlyChanges
              ? 'bg-indigo-600 text-white font-medium'
              : 'bg-slate-800 hover:bg-slate-700 text-slate-300'
          }`}
        >
          <Filter className="w-3.5 h-3.5" />
          {onlyChanges ? 'Showing Changed Only' : 'Show All Keys'}
        </button>
      </div>

      {/* Side-by-side Diff View */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Before Column */}
        <div className="flex flex-col bg-slate-950 border border-slate-800 rounded-lg overflow-hidden">
          <div className="flex items-center justify-between px-3 py-2 bg-slate-900 border-b border-slate-800 text-xs font-medium text-slate-300">
            <span>{beforeTitle}</span>
            <button
              onClick={() => copyToClipboard(JSON.stringify(beforeState, null, 2), true)}
              className="flex items-center gap-1 text-slate-400 hover:text-slate-200 transition-colors"
              title="Copy JSON"
            >
              {copiedBefore ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
              <span>{copiedBefore ? 'Copied' : 'Copy'}</span>
            </button>
          </div>
          <div className="p-3 divide-y divide-slate-800/60 max-h-[500px] overflow-y-auto font-mono text-xs">
            {displayedItems.length === 0 ? (
              <div className="text-slate-500 py-4 text-center">No keys to display</div>
            ) : (
              displayedItems.map((item) => (
                <div key={'before-' + item.key} className="py-2">
                  <div className="flex items-center justify-between mb-1">
                    <span className="font-semibold text-slate-200">{item.key}</span>
                    {item.status === 'removed' && (
                      <span className="px-1.5 py-0.5 text-[10px] rounded bg-rose-950 text-rose-300 border border-rose-800/40">
                        REMOVED
                      </span>
                    )}
                    {item.status === 'modified' && (
                      <span className="px-1.5 py-0.5 text-[10px] rounded bg-amber-950 text-amber-300 border border-amber-800/40">
                        MODIFIED
                      </span>
                    )}
                  </div>
                  {item.jsonBefore !== undefined ? (
                    <pre
                      className={`p-2 rounded overflow-x-auto ${
                        item.status === 'removed'
                          ? 'bg-rose-950/20 text-rose-200 border border-rose-900/30'
                          : item.status === 'modified'
                          ? 'bg-amber-950/20 text-amber-200 border border-amber-900/30'
                          : 'bg-slate-900/50 text-slate-400'
                      }`}
                    >
                      {item.jsonBefore}
                    </pre>
                  ) : (
                    <div className="italic text-slate-600 px-2 py-1">undefined (not present)</div>
                  )}
                </div>
              ))
            )}
          </div>
        </div>

        {/* After Column */}
        <div className="flex flex-col bg-slate-950 border border-slate-800 rounded-lg overflow-hidden">
          <div className="flex items-center justify-between px-3 py-2 bg-slate-900 border-b border-slate-800 text-xs font-medium text-slate-300">
            <span>{afterTitle}</span>
            <button
              onClick={() => copyToClipboard(JSON.stringify(afterState, null, 2), false)}
              className="flex items-center gap-1 text-slate-400 hover:text-slate-200 transition-colors"
              title="Copy JSON"
            >
              {copiedAfter ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
              <span>{copiedAfter ? 'Copied' : 'Copy'}</span>
            </button>
          </div>
          <div className="p-3 divide-y divide-slate-800/60 max-h-[500px] overflow-y-auto font-mono text-xs">
            {displayedItems.length === 0 ? (
              <div className="text-slate-500 py-4 text-center">No keys to display</div>
            ) : (
              displayedItems.map((item) => (
                <div key={'after-' + item.key} className="py-2">
                  <div className="flex items-center justify-between mb-1">
                    <span className="font-semibold text-slate-200">{item.key}</span>
                    {item.status === 'added' && (
                      <span className="px-1.5 py-0.5 text-[10px] rounded bg-emerald-950 text-emerald-300 border border-emerald-800/40">
                        + ADDED
                      </span>
                    )}
                    {item.status === 'modified' && (
                      <span className="px-1.5 py-0.5 text-[10px] rounded bg-amber-950 text-amber-300 border border-amber-800/40">
                        MODIFIED
                      </span>
                    )}
                  </div>
                  {item.jsonAfter !== undefined ? (
                    <pre
                      className={`p-2 rounded overflow-x-auto ${
                        item.status === 'added'
                          ? 'bg-emerald-950/20 text-emerald-200 border border-emerald-900/30'
                          : item.status === 'modified'
                          ? 'bg-amber-950/20 text-amber-200 border border-amber-900/30'
                          : 'bg-slate-900/50 text-slate-400'
                      }`}
                    >
                      {item.jsonAfter}
                    </pre>
                  ) : (
                    <div className="italic text-slate-600 px-2 py-1">undefined (removed)</div>
                  )}
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
