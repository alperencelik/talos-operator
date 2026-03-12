import React from 'react';
import { Server, Cpu, Layers, HardDrive, Wand2, GitBranch, ArrowRight } from 'lucide-react';
import { Page, Resources } from '../App';

interface StatCardProps {
  label: string;
  count: number;
  icon: React.ReactNode;
  color: string;
  onClick: () => void;
}

function StatCard({ label, count, icon, color, onClick }: StatCardProps) {
  return (
    <button
      onClick={onClick}
      className="bg-zinc-900 border border-zinc-800 rounded-xl p-5 text-left hover:border-zinc-700 hover:bg-zinc-800/50 transition-all group"
    >
      <div className="flex items-start justify-between">
        <div className={`p-2 rounded-lg ${color}`}>{icon}</div>
        <ArrowRight size={14} className="text-zinc-600 group-hover:text-zinc-400 transition-colors mt-1" />
      </div>
      <div className="mt-4">
        <div className="text-3xl font-semibold text-zinc-100">{count}</div>
        <div className="text-sm text-zinc-500 mt-0.5">{label}</div>
      </div>
    </button>
  );
}

interface OverviewProps {
  resources: Resources | null;
  loading: boolean;
  onNavigate: (page: Page) => void;
}

export default function Overview({ resources, loading, onNavigate }: OverviewProps) {
  const counts = {
    clusters: resources?.talosClusters?.length ?? 0,
    controlPlanes: resources?.talosControlPlanes?.length ?? 0,
    workers: resources?.talosWorkers?.length ?? 0,
    machines: resources?.talosMachines?.length ?? 0,
  };

  const total = counts.clusters + counts.controlPlanes + counts.workers + counts.machines;

  return (
    <div className="p-6 max-w-5xl">
      {/* Page header */}
      <div className="mb-8">
        <h1 className="text-xl font-semibold text-zinc-100">Overview</h1>
        <p className="text-sm text-zinc-500 mt-1">
          Monitor and manage your Talos Linux clusters
        </p>
      </div>

      {/* Stat cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
        <StatCard
          label="Clusters"
          count={counts.clusters}
          icon={<Server size={16} className="text-brand" />}
          color="bg-brand-dim"
          onClick={() => onNavigate('clusters')}
        />
        <StatCard
          label="Control Planes"
          count={counts.controlPlanes}
          icon={<Cpu size={16} className="text-sky-400" />}
          color="bg-sky-950"
          onClick={() => onNavigate('control-planes')}
        />
        <StatCard
          label="Workers"
          count={counts.workers}
          icon={<Layers size={16} className="text-purple-400" />}
          color="bg-purple-950"
          onClick={() => onNavigate('workers')}
        />
        <StatCard
          label="Machines"
          count={counts.machines}
          icon={<HardDrive size={16} className="text-emerald-400" />}
          color="bg-emerald-950"
          onClick={() => onNavigate('machines')}
        />
      </div>

      {/* Content area */}
      {loading ? (
        <div className="flex items-center gap-3 text-sm text-zinc-500 py-8">
          <div className="w-4 h-4 border-2 border-zinc-700 border-t-brand rounded-full animate-spin" />
          Loading resources…
        </div>
      ) : total === 0 ? (
        /* Empty state */
        <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-10 text-center">
          <div className="w-12 h-12 bg-brand-dim rounded-xl flex items-center justify-center mx-auto mb-4">
            <Server size={20} className="text-brand" />
          </div>
          <h2 className="text-base font-semibold text-zinc-100 mb-1">No resources found</h2>
          <p className="text-sm text-zinc-500 mb-6 max-w-sm mx-auto">
            No Talos resources were detected in the cluster. Use the Generator to create your first cluster.
          </p>
          <div className="flex items-center justify-center gap-3">
            <button
              onClick={() => onNavigate('generator')}
              className="flex items-center gap-2 px-4 py-2 bg-brand text-white rounded-lg text-sm font-medium hover:bg-brand-hover transition-colors"
            >
              <Wand2 size={14} />
              Open Generator
            </button>
            <button
              onClick={() => onNavigate('visualizer')}
              className="flex items-center gap-2 px-4 py-2 bg-zinc-800 text-zinc-300 rounded-lg text-sm font-medium hover:bg-zinc-700 transition-colors"
            >
              <GitBranch size={14} />
              Open Visualizer
            </button>
          </div>
        </div>
      ) : (
        /* Recent resources */
        <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden">
          <div className="px-5 py-4 border-b border-zinc-800 flex items-center justify-between">
            <h2 className="text-sm font-semibold text-zinc-200">All Resources</h2>
            <span className="text-xs text-zinc-500">{total} total</span>
          </div>
          <div className="divide-y divide-zinc-800">
            {[
              ...( resources?.talosClusters ?? []).map(r => ({ ...r, _kind: 'TalosCluster', _page: 'clusters' as Page })),
              ...( resources?.talosControlPlanes ?? []).map(r => ({ ...r, _kind: 'TalosControlPlane', _page: 'control-planes' as Page })),
              ...( resources?.talosWorkers ?? []).map(r => ({ ...r, _kind: 'TalosWorker', _page: 'workers' as Page })),
              ...( resources?.talosMachines ?? []).map(r => ({ ...r, _kind: 'TalosMachine', _page: 'machines' as Page })),
            ].slice(0, 10).map((item, i) => (
              <div key={i} className="px-5 py-3 flex items-center justify-between hover:bg-zinc-800/40 transition-colors">
                <div className="flex items-center gap-3 min-w-0">
                  <KindBadge kind={item._kind} />
                  <div className="min-w-0">
                    <span className="text-sm text-zinc-200 font-medium truncate block">{item.metadata?.name}</span>
                    <span className="text-xs text-zinc-600">{item.metadata?.namespace}</span>
                  </div>
                </div>
                <button
                  onClick={() => onNavigate(item._page)}
                  className="text-xs text-zinc-500 hover:text-zinc-300 transition-colors flex-shrink-0 ml-4"
                >
                  View →
                </button>
              </div>
            ))}
          </div>
          {total > 10 && (
            <div className="px-5 py-3 border-t border-zinc-800 text-xs text-zinc-600">
              Showing 10 of {total} resources
            </div>
          )}
        </div>
      )}

      {/* Quick actions */}
      <div className="mt-6 flex items-center gap-3">
        <button
          onClick={() => onNavigate('generator')}
          className="flex items-center gap-2 px-3 py-2 bg-zinc-900 border border-zinc-700 text-zinc-300 rounded-lg text-sm hover:bg-zinc-800 transition-colors"
        >
          <Wand2 size={13} />
          Generator
        </button>
        <button
          onClick={() => onNavigate('visualizer')}
          className="flex items-center gap-2 px-3 py-2 bg-zinc-900 border border-zinc-700 text-zinc-300 rounded-lg text-sm hover:bg-zinc-800 transition-colors"
        >
          <GitBranch size={13} />
          Visualizer
        </button>
      </div>
    </div>
  );
}

function KindBadge({ kind }: { kind: string }) {
  const styles: Record<string, string> = {
    TalosCluster: 'bg-brand-dim text-brand',
    TalosControlPlane: 'bg-sky-950 text-sky-400',
    TalosWorker: 'bg-purple-950 text-purple-400',
    TalosMachine: 'bg-emerald-950 text-emerald-400',
  };
  const shortName: Record<string, string> = {
    TalosCluster: 'Cluster',
    TalosControlPlane: 'CP',
    TalosWorker: 'Worker',
    TalosMachine: 'Machine',
  };
  return (
    <span className={`inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium ${styles[kind] ?? 'bg-zinc-800 text-zinc-400'}`}>
      {shortName[kind] ?? kind}
    </span>
  );
}
