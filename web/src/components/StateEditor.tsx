import React, { useState } from 'react';
import { Check, Copy, Edit3, AlertCircle } from 'lucide-react';

interface StateEditorProps {
  initialState: Record<string, any>;
  onSave?: (updatedState: Record<string, any>) => void;
  readOnly?: boolean;
}

export const StateEditor: React.FC<StateEditorProps> = ({
  initialState,
  onSave,
  readOnly = false,
}) => {
  const [isEditing, setIsEditing] = useState(false);
  const [rawText, setRawText] = useState(JSON.stringify(initialState, null, 2));
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const handleTextChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const val = e.target.value;
    setRawText(val);
    try {
      JSON.parse(val);
      setError(null);
    } catch (err: any) {
      setError(err.message);
    }
  };

  const handleApply = () => {
    try {
      const parsed = JSON.parse(rawText);
      setError(null);
      setIsEditing(false);
      if (onSave) {
        onSave(parsed);
      }
    } catch (err: any) {
      setError('Cannot apply: ' + err.message);
    }
  };

  const handleCopy = () => {
    navigator.clipboard.writeText(rawText);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="bg-slate-900/60 border border-slate-800 rounded-xl overflow-hidden flex flex-col">
      <div className="flex items-center justify-between px-4 py-3 bg-slate-950/80 border-b border-slate-800">
        <div className="flex items-center gap-2">
          <span className="text-xs font-semibold text-slate-300">Execution Memory State</span>
          <span className="text-[11px] font-mono text-slate-500">JSON</span>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={handleCopy}
            className="px-2 py-1 rounded text-xs text-slate-400 hover:text-slate-200 hover:bg-slate-800 transition-colors flex items-center gap-1"
          >
            {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
            {copied ? 'Copied' : 'Copy'}
          </button>

          {!readOnly && (
            <button
              onClick={() => {
                if (isEditing) {
                  handleApply();
                } else {
                  setIsEditing(true);
                }
              }}
              disabled={isEditing && !!error}
              className={`px-2.5 py-1 rounded text-xs font-medium flex items-center gap-1 transition-colors ${
                isEditing
                  ? error
                    ? 'bg-slate-800 text-slate-500 cursor-not-allowed'
                    : 'bg-cyan-500 text-slate-950 hover:bg-cyan-400 font-semibold'
                  : 'bg-slate-800 text-slate-300 hover:bg-slate-700'
              }`}
            >
              {isEditing ? (
                <>
                  <Check className="w-3.5 h-3.5" />
                  Save Edits
                </>
              ) : (
                <>
                  <Edit3 className="w-3.5 h-3.5" />
                  Edit State
                </>
              )}
            </button>
          )}
        </div>
      </div>

      <div className="p-4 relative">
        {isEditing ? (
          <div>
            <textarea
              value={rawText}
              onChange={handleTextChange}
              rows={16}
              className="w-full bg-slate-950 border border-slate-800 rounded-lg p-3 text-xs font-mono text-cyan-200 focus:outline-none focus:border-cyan-500/80 resize-y"
            />
            {error && (
              <div className="mt-2 text-xs text-rose-400 flex items-center gap-1.5 font-mono">
                <AlertCircle className="w-3.5 h-3.5" />
                {error}
              </div>
            )}
          </div>
        ) : (
          <pre className="p-3 bg-slate-950/70 border border-slate-800/80 rounded-lg text-xs font-mono text-slate-200 overflow-x-auto max-h-[460px] leading-relaxed">
            {rawText}
          </pre>
        )}
      </div>
    </div>
  );
};
