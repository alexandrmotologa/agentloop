import React from 'react';
import { Activity, Play, Inbox, GitBranch, RefreshCw, Layers } from 'lucide-react';

interface HeaderProps {
  activeTab: string;
  setActiveTab: (tab: string) => void;
  pendingDLQCount: number;
  onOpenTrigger: () => void;
  onRefresh: () => void;
}

export const Header: React.FC<HeaderProps> = ({
  activeTab,
  setActiveTab,
  pendingDLQCount,
  onOpenTrigger,
  onRefresh,
}) => {
  return (
    <>
      <header className="border-b border-slate-800/80 bg-slate-950/80 backdrop-blur-md sticky top-0 z-40">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
          {/* Brand */}
          <div className="flex items-center gap-6">
            <div 
              onClick={() => setActiveTab('dashboard')} 
              className="flex items-center gap-3 cursor-pointer group"
            >
              <div className="w-9 h-9 rounded-lg bg-gradient-to-tr from-cyan-600 to-blue-500 flex items-center justify-center shadow-lg shadow-cyan-500/20 group-hover:scale-105 transition-transform">
                <RefreshCw className="w-5 h-5 text-white" />
              </div>
              <div>
                <div className="flex items-center gap-2">
                  <span className="font-bold text-lg tracking-tight text-white">agentloop</span>
                  <span className="px-1.5 py-0.5 text-[10px] font-mono font-medium rounded bg-cyan-950/80 text-cyan-400 border border-cyan-800/50">
                    ENGINE
                  </span>
                </div>
                <p className="text-[11px] text-slate-400 font-normal">Self-Healing HITL Cockpit</p>
              </div>
            </div>

            {/* Desktop Navigation Links */}
            <nav className="hidden md:flex items-center gap-1 pl-4 border-l border-slate-800">
              <button
                onClick={() => setActiveTab('dashboard')}
                className={`px-3 py-1.5 rounded-md text-sm font-medium transition-colors flex items-center gap-2 ${
                  activeTab === 'dashboard'
                    ? 'bg-slate-800/80 text-cyan-400 border border-slate-700/60'
                    : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/40'
                }`}
              >
                <Activity className="w-4 h-4" />
                Dashboard
              </button>
              <button
                onClick={() => setActiveTab('executions')}
                className={`px-3 py-1.5 rounded-md text-sm font-medium transition-colors flex items-center gap-2 ${
                  activeTab === 'executions'
                    ? 'bg-slate-800/80 text-cyan-400 border border-slate-700/60'
                    : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/40'
                }`}
              >
                <Layers className="w-4 h-4" />
                Executions
              </button>
              <button
                onClick={() => setActiveTab('dlq')}
                className={`px-3 py-1.5 rounded-md text-sm font-medium transition-colors flex items-center gap-2 relative ${
                  activeTab === 'dlq'
                    ? 'bg-slate-800/80 text-cyan-400 border border-slate-700/60'
                    : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/40'
                }`}
              >
                <Inbox className="w-4 h-4" />
                DLQ Review
                {pendingDLQCount > 0 && (
                  <span className="px-1.5 py-0.2 text-[11px] font-mono font-bold rounded-full bg-amber-500/20 text-amber-300 border border-amber-500/40">
                    {pendingDLQCount}
                  </span>
                )}
              </button>
              <button
                onClick={() => setActiveTab('workflows')}
                className={`px-3 py-1.5 rounded-md text-sm font-medium transition-colors flex items-center gap-2 ${
                  activeTab === 'workflows'
                    ? 'bg-slate-800/80 text-cyan-400 border border-slate-700/60'
                    : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/40'
                }`}
              >
                <GitBranch className="w-4 h-4" />
                Specs
              </button>
            </nav>
          </div>

          {/* Action Buttons */}
          <div className="flex items-center gap-2 sm:gap-3">
            <button
              onClick={onRefresh}
              title="Refresh data"
              className="p-2 rounded-lg text-slate-400 hover:text-slate-200 hover:bg-slate-800/60 transition-colors"
            >
              <RefreshCw className="w-4 h-4" />
            </button>
            <button
              onClick={onOpenTrigger}
              className="px-3 sm:px-3.5 py-1.5 rounded-lg text-xs sm:text-sm bg-gradient-to-r from-cyan-500 to-blue-600 hover:from-cyan-400 hover:to-blue-500 text-slate-950 font-semibold shadow-lg shadow-cyan-500/10 flex items-center gap-1.5 sm:gap-2 transition-all active:scale-95"
            >
              <Play className="w-3.5 h-3.5 fill-slate-950" />
              <span>Trigger</span>
            </button>
          </div>
        </div>
      </header>

      {/* Mobile Bottom Navigation Bar (Thumb-friendly responsive bar for < md) */}
      <nav className="fixed bottom-0 left-0 right-0 z-50 md:hidden bg-slate-950/95 border-t border-slate-800 backdrop-blur-lg flex items-center justify-around py-2 px-2 shadow-2xl">
        <button
          onClick={() => setActiveTab('dashboard')}
          className={`flex flex-col items-center gap-1 px-3 py-1 rounded-lg text-[11px] font-medium transition-colors ${
            activeTab === 'dashboard' ? 'text-cyan-400' : 'text-slate-400 hover:text-slate-200'
          }`}
        >
          <Activity className="w-4 h-4" />
          <span>Overview</span>
        </button>
        <button
          onClick={() => setActiveTab('executions')}
          className={`flex flex-col items-center gap-1 px-3 py-1 rounded-lg text-[11px] font-medium transition-colors ${
            activeTab === 'executions' ? 'text-cyan-400' : 'text-slate-400 hover:text-slate-200'
          }`}
        >
          <Layers className="w-4 h-4" />
          <span>Runs</span>
        </button>
        <button
          onClick={() => setActiveTab('dlq')}
          className={`flex flex-col items-center gap-1 px-3 py-1 rounded-lg text-[11px] font-medium transition-colors relative ${
            activeTab === 'dlq' ? 'text-cyan-400' : 'text-slate-400 hover:text-slate-200'
          }`}
        >
          <Inbox className="w-4 h-4" />
          <span>DLQ</span>
          {pendingDLQCount > 0 && (
            <span className="absolute top-0 right-2 w-2 h-2 rounded-full bg-amber-400 animate-pulse"></span>
          )}
        </button>
        <button
          onClick={() => setActiveTab('workflows')}
          className={`flex flex-col items-center gap-1 px-3 py-1 rounded-lg text-[11px] font-medium transition-colors ${
            activeTab === 'workflows' ? 'text-cyan-400' : 'text-slate-400 hover:text-slate-200'
          }`}
        >
          <GitBranch className="w-4 h-4" />
          <span>Specs</span>
        </button>
      </nav>
    </>
  );
};
