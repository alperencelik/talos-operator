import React from 'react';
import {
  Server,
  Cpu,
  Layers,
  HardDrive,
  Wand2,
  GitBranch,
  ArrowRight,
  Package,
  Tag,
  Database,
  CalendarClock,
} from 'lucide-react';
import { Link } from 'react-router-dom';
import { KIND_PATH, Resources } from '../App';

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
  onNavigate: (path: string) => void;
}

interface MiniTileProps {
  label: string;
  count: number;
  icon: React.ReactNode;
  onClick: () => void;
}

function MiniTile({ label, count, icon, onClick }: MiniTileProps) {
  return (
    <button
      onClick={onClick}
      className="flex items-center gap-3 bg-zinc-900 border border-zinc-800 rounded-lg px-4 py-2.5 hover:border-zinc-700 hover:bg-zinc-800/50 transition-all text-left"
    >
      <span className="text-zinc-500">{icon}</span>
      <div className="flex items-baseline gap-1.5">
        <span className="text-base font-semibold text-zinc-100">{count}</span>
        <span className="text-xs text-zinc-500">{label}</span>
      </div>
    </button>
  );
}

export default function Overview({ resources, loading, onNavigate }: OverviewProps) {
  const counts = {
    clusters: resources?.talosClusters?.length ?? 0,
    controlPlanes: resources?.talosControlPlanes?.length ?? 0,
    workers: resources?.talosWorkers?.length ?? 0,
    machines: resources?.talosMachines?.length ?? 0,
    addons: resources?.talosClusterAddons?.length ?? 0,
    addonReleases: resources?.talosClusterAddonReleases?.length ?? 0,
    etcdBackups: resources?.talosEtcdBackups?.length ?? 0,
    etcdBackupSchedules: resources?.talosEtcdBackupSchedules?.length ?? 0,
  };

  const total =
    counts.clusters +
    counts.controlPlanes +
    counts.workers +
    counts.machines +
    counts.addons +
    counts.addonReleases +
    counts.etcdBackups +
    counts.etcdBackupSchedules;

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
          onClick={() => onNavigate('/clusters')}
        />
        <StatCard
          label="Control Planes"
          count={counts.controlPlanes}
          icon={<Cpu size={16} className="text-sky-400" />}
          color="bg-sky-950"
          onClick={() => onNavigate('/control-planes')}
        />
        <StatCard
          label="Workers"
          count={counts.workers}
          icon={<Layers size={16} className="text-purple-400" />}
          color="bg-purple-950"
          onClick={() => onNavigate('/workers')}
        />
        <StatCard
          label="Machines"
          count={counts.machines}
          icon={<HardDrive size={16} className="text-emerald-400" />}
          color="bg-emerald-950"
          onClick={() => onNavigate('/machines')}
        />
      </div>

      {/* Secondary resources */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 mb-8">
        <MiniTile
          label="Add-ons"
          count={counts.addons}
          icon={<Package size={14} />}
          onClick={() => onNavigate('/addons')}
        />
        <MiniTile
          label="Releases"
          count={counts.addonReleases}
          icon={<Tag size={14} />}
          onClick={() => onNavigate('/addon-releases')}
        />
        <MiniTile
          label="Etcd Backups"
          count={counts.etcdBackups}
          icon={<Database size={14} />}
          onClick={() => onNavigate('/etcd-backups')}
        />
        <MiniTile
          label="Schedules"
          count={counts.etcdBackupSchedules}
          icon={<CalendarClock size={14} />}
          onClick={() => onNavigate('/etcd-backup-schedules')}
        />
      </div>

      {/* Per-cluster health */}
      {!loading && (resources?.talosClusters?.length ?? 0) > 0 && (
        <ClusterHealth resources={resources!} onNavigate={onNavigate} />
      )}

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
              onClick={() => onNavigate('/generator')}
              className="flex items-center gap-2 px-4 py-2 bg-brand text-white rounded-lg text-sm font-medium hover:bg-brand-hover transition-colors"
            >
              <Wand2 size={14} />
              Open Generator
            </button>
            <button
              onClick={() => onNavigate('/visualizer')}
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
              ...(resources?.talosClusters ?? []).map(r => ({ ...r, _kind: 'TalosCluster' })),
              ...(resources?.talosControlPlanes ?? []).map(r => ({ ...r, _kind: 'TalosControlPlane' })),
              ...(resources?.talosWorkers ?? []).map(r => ({ ...r, _kind: 'TalosWorker' })),
              ...(resources?.talosMachines ?? []).map(r => ({ ...r, _kind: 'TalosMachine' })),
              ...(resources?.talosClusterAddons ?? []).map(r => ({ ...r, _kind: 'TalosClusterAddon' })),
              ...(resources?.talosClusterAddonReleases ?? []).map(r => ({ ...r, _kind: 'TalosClusterAddonRelease' })),
              ...(resources?.talosEtcdBackups ?? []).map(r => ({ ...r, _kind: 'TalosEtcdBackup' })),
              ...(resources?.talosEtcdBackupSchedules ?? []).map(r => ({ ...r, _kind: 'TalosEtcdBackupSchedule' })),
            ].slice(0, 10).map((item, i) => {
              const detailPath = `${KIND_PATH[item._kind]}/${item.metadata?.namespace}/${item.metadata?.name}`;
              return (
                <div key={i} className="px-5 py-3 flex items-center justify-between hover:bg-zinc-800/40 transition-colors">
                  <div className="flex items-center gap-3 min-w-0">
                    <KindBadge kind={item._kind} />
                    <div className="min-w-0">
                      <Link
                        to={detailPath}
                        className="text-sm text-zinc-200 font-medium truncate block hover:text-brand transition-colors"
                      >
                        {item.metadata?.name}
                      </Link>
                      <span className="text-xs text-zinc-600">{item.metadata?.namespace}</span>
                    </div>
                  </div>
                  <Link
                    to={detailPath}
                    className="text-xs text-zinc-500 hover:text-zinc-300 transition-colors flex-shrink-0 ml-4"
                  >
                    View →
                  </Link>
                </div>
              );
            })}
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
          onClick={() => onNavigate('/generator')}
          className="flex items-center gap-2 px-3 py-2 bg-zinc-900 border border-zinc-700 text-zinc-300 rounded-lg text-sm hover:bg-zinc-800 transition-colors"
        >
          <Wand2 size={13} />
          Generator
        </button>
        <button
          onClick={() => onNavigate('/visualizer')}
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
    TalosClusterAddon: 'bg-indigo-950 text-indigo-400',
    TalosClusterAddonRelease: 'bg-indigo-950 text-indigo-400',
    TalosEtcdBackup: 'bg-amber-950 text-amber-400',
    TalosEtcdBackupSchedule: 'bg-amber-950 text-amber-400',
  };
  const shortName: Record<string, string> = {
    TalosCluster: 'Cluster',
    TalosControlPlane: 'CP',
    TalosWorker: 'Worker',
    TalosMachine: 'Machine',
    TalosClusterAddon: 'Add-on',
    TalosClusterAddonRelease: 'Release',
    TalosEtcdBackup: 'Backup',
    TalosEtcdBackupSchedule: 'Schedule',
  };
  return (
    <span className={`inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium ${styles[kind] ?? 'bg-zinc-800 text-zinc-400'}`}>
      {shortName[kind] ?? kind}
    </span>
  );
}

// ─── Cluster health tile ──────────────────────────────────────────────────────

function isReady(item: any): boolean {
  const c = item?.status?.conditions?.find((cond: any) => cond.type === 'Ready');
  return c?.status === 'True';
}

function ageLabel(ts?: string): string {
  if (!ts) return '—';
  const ms = Date.now() - new Date(ts).getTime();
  if (ms < 0) return 'just now';
  const m = Math.floor(ms / 60000);
  const h = Math.floor(m / 60);
  const d = Math.floor(h / 24);
  if (d > 0) return `${d}d ago`;
  if (h > 0) return `${h}h ago`;
  if (m > 0) return `${m}m ago`;
  return 'just now';
}

interface ClusterHealthRow {
  cluster: any;
  readyCount: number;
  totalCount: number;
  lastBackup?: string;
}

function computeHealth(cluster: any, resources: Resources): ClusterHealthRow {
  const clusterName = cluster.metadata?.name;
  const ownedBy = (item: any, kind: string, name: string) =>
    (item.metadata?.ownerReferences ?? []).some((r: any) => r.kind === kind && r.name === name);

  const cps = resources.talosControlPlanes.filter(cp => ownedBy(cp, 'TalosCluster', clusterName));
  const workers = resources.talosWorkers.filter(w => ownedBy(w, 'TalosCluster', clusterName));
  const cpNames = new Set(cps.map(cp => cp.metadata?.name));
  const workerNames = new Set(workers.map(w => w.metadata?.name));
  const machines = resources.talosMachines.filter(
    m =>
      (m.metadata?.ownerReferences ?? []).some(
        (r: any) =>
          (r.kind === 'TalosControlPlane' && cpNames.has(r.name)) ||
          (r.kind === 'TalosWorker' && workerNames.has(r.name))
      )
  );

  const all = [cluster, ...cps, ...workers, ...machines];
  const ready = all.filter(isReady).length;

  // Most recent successful etcd backup whose spec points at one of this
  // cluster's control planes.
  const backups = resources.talosEtcdBackups.filter(
    b => cpNames.has(b.spec?.talosControlPlaneRef?.name) && isReady(b)
  );
  backups.sort(
    (a, b) =>
      new Date(b.metadata?.creationTimestamp ?? 0).getTime() -
      new Date(a.metadata?.creationTimestamp ?? 0).getTime()
  );

  return {
    cluster,
    readyCount: ready,
    totalCount: all.length,
    lastBackup: backups[0]?.metadata?.creationTimestamp,
  };
}

function ClusterHealth({
  resources,
  onNavigate,
}: {
  resources: Resources;
  onNavigate: (path: string) => void;
}) {
  const healths = resources.talosClusters.map(c => computeHealth(c, resources));

  return (
    <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden mb-8">
      <div className="px-5 py-4 border-b border-zinc-800 flex items-center justify-between">
        <h2 className="text-sm font-semibold text-zinc-200">Cluster Health</h2>
        <span className="text-xs text-zinc-500">{healths.length} clusters</span>
      </div>
      <div className="divide-y divide-zinc-800">
        {healths.map(h => {
          const pct = h.totalCount > 0 ? Math.round((h.readyCount / h.totalCount) * 100) : 0;
          const tone =
            pct === 100 ? 'green' : pct >= 75 ? 'yellow' : pct >= 1 ? 'red' : 'gray';
          const barColor = {
            green: 'bg-green-500',
            yellow: 'bg-yellow-500',
            red: 'bg-red-500',
            gray: 'bg-zinc-600',
          }[tone];
          const textColor = {
            green: 'text-green-400',
            yellow: 'text-yellow-400',
            red: 'text-red-400',
            gray: 'text-zinc-500',
          }[tone];
          return (
            <button
              key={h.cluster.metadata?.uid ?? h.cluster.metadata?.name}
              onClick={() =>
                onNavigate(
                  `/clusters/${h.cluster.metadata?.namespace}/${h.cluster.metadata?.name}`
                )
              }
              className="w-full px-5 py-3 flex items-center gap-5 hover:bg-zinc-800/40 transition-colors text-left"
            >
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span className="text-sm text-zinc-100 font-medium truncate">
                    {h.cluster.metadata?.name}
                  </span>
                  <span className="text-xs text-zinc-600">{h.cluster.metadata?.namespace}</span>
                </div>
                <div className="mt-1.5 flex items-center gap-2">
                  <div className="flex-1 h-1 bg-zinc-800 rounded-full overflow-hidden max-w-xs">
                    <div className={`h-full ${barColor}`} style={{ width: `${pct}%` }} />
                  </div>
                  <span className={`text-xs ${textColor} flex-shrink-0`}>
                    {h.readyCount}/{h.totalCount} ready · {pct}%
                  </span>
                </div>
              </div>
              <div className="flex-shrink-0 text-right">
                <div className="text-[10px] uppercase tracking-wider text-zinc-600">
                  Last backup
                </div>
                <div className="text-xs text-zinc-400 mt-0.5">{ageLabel(h.lastBackup)}</div>
              </div>
            </button>
          );
        })}
      </div>
    </div>
  );
}
