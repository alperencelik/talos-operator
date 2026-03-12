import React, { useState } from 'react';
import { FileCode, X, ChevronRight } from 'lucide-react';
import YAML from 'js-yaml';

interface ResourceListProps {
  title: string;
  kind: string;
  items: any[];
  loading: boolean;
}

function getStatus(item: any): { label: string; className: string } {
  const phase = item.status?.phase;
  if (phase) {
    const lower = phase.toLowerCase();
    if (['ready', 'running', 'active', 'succeeded'].includes(lower))
      return { label: phase, className: 'text-green-400' };
    if (['failed', 'error', 'crashloopbackoff'].includes(lower))
      return { label: phase, className: 'text-red-400' };
    if (['pending', 'creating', 'initializing', 'provisioning'].includes(lower))
      return { label: phase, className: 'text-yellow-400' };
    return { label: phase, className: 'text-zinc-400' };
  }

  const conditions: any[] = item.status?.conditions ?? [];
  const ready = conditions.find((c: any) => c.type === 'Ready');
  if (ready) {
    return ready.status === 'True'
      ? { label: 'Ready', className: 'text-green-400' }
      : { label: 'Not Ready', className: 'text-red-400' };
  }

  return { label: '—', className: 'text-zinc-600' };
}

function getAge(timestamp?: string): string {
  if (!timestamp) return '—';
  const diff = Date.now() - new Date(timestamp).getTime();
  const minutes = Math.floor(diff / 60000);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);
  if (days > 0) return `${days}d`;
  if (hours > 0) return `${hours}h`;
  if (minutes > 0) return `${minutes}m`;
  return 'just now';
}

function YamlModal({ item, onClose }: { item: any; onClose: () => void }) {
  const yaml = YAML.dump(item);
  return (
    <div
      className="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center z-50 p-4"
      onClick={onClose}
    >
      <div
        className="bg-zinc-900 border border-zinc-700 rounded-xl shadow-2xl w-full max-w-2xl max-h-[85vh] flex flex-col"
        onClick={e => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-zinc-800 flex-shrink-0">
          <div className="flex items-center gap-2.5 min-w-0">
            <FileCode size={15} className="text-zinc-500 flex-shrink-0" />
            <span className="text-sm font-semibold text-zinc-100 truncate">
              {item?.metadata?.name}
            </span>
            <span className="text-xs text-zinc-500 flex-shrink-0">
              {item?.metadata?.namespace}
            </span>
          </div>
          <button
            onClick={onClose}
            className="text-zinc-500 hover:text-zinc-300 transition-colors flex-shrink-0 ml-3 p-1 hover:bg-zinc-800 rounded"
          >
            <X size={15} />
          </button>
        </div>

        {/* YAML content */}
        <div className="overflow-auto flex-1 p-5">
          <pre className="text-xs font-mono text-zinc-300 leading-relaxed whitespace-pre-wrap">{yaml}</pre>
        </div>
      </div>
    </div>
  );
}

export default function ResourceList({ title, kind, items, loading }: ResourceListProps) {
  const [selectedItem, setSelectedItem] = useState<any>(null);

  return (
    <div className="p-6">
      {/* Page header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-zinc-100">{title}</h1>
          <p className="text-sm text-zinc-500 mt-0.5">{kind} resources in the cluster</p>
        </div>
        {!loading && (
          <span className="text-xs text-zinc-600 bg-zinc-900 border border-zinc-800 px-2.5 py-1 rounded-full">
            {items.length} {items.length === 1 ? 'resource' : 'resources'}
          </span>
        )}
      </div>

      {/* Table */}
      <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden">
        {loading ? (
          <div className="flex items-center gap-3 text-sm text-zinc-500 p-8">
            <div className="w-4 h-4 border-2 border-zinc-700 border-t-brand rounded-full animate-spin" />
            Loading…
          </div>
        ) : items.length === 0 ? (
          <div className="text-center py-16 px-8">
            <div className="text-zinc-600 text-sm mb-1">No {title.toLowerCase()} found</div>
            <div className="text-zinc-700 text-xs">
              Resources will appear here once they are created in the cluster.
            </div>
          </div>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-zinc-800">
                <th className="px-5 py-3 text-left text-xs font-medium text-zinc-500 uppercase tracking-wider">Name</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-zinc-500 uppercase tracking-wider">Namespace</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-zinc-500 uppercase tracking-wider">Status</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-zinc-500 uppercase tracking-wider">Age</th>
                <th className="px-5 py-3 text-right text-xs font-medium text-zinc-500 uppercase tracking-wider">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-800">
              {items.map((item, i) => {
                const status = getStatus(item);
                const age = getAge(item.metadata?.creationTimestamp);
                return (
                  <tr key={i} className="hover:bg-zinc-800/30 transition-colors">
                    <td className="px-5 py-3.5">
                      <span className="font-medium text-zinc-200">{item.metadata?.name ?? '—'}</span>
                    </td>
                    <td className="px-5 py-3.5">
                      <span className="text-zinc-500">{item.metadata?.namespace ?? '—'}</span>
                    </td>
                    <td className="px-5 py-3.5">
                      <div className="flex items-center gap-2">
                        <span className={`w-1.5 h-1.5 rounded-full flex-shrink-0 ${
                          status.className.includes('green') ? 'bg-green-500' :
                          status.className.includes('red') ? 'bg-red-500' :
                          status.className.includes('yellow') ? 'bg-yellow-500' : 'bg-zinc-600'
                        }`} />
                        <span className={`text-xs ${status.className}`}>{status.label}</span>
                      </div>
                    </td>
                    <td className="px-5 py-3.5">
                      <span className="text-zinc-500 text-xs">{age}</span>
                    </td>
                    <td className="px-5 py-3.5 text-right">
                      <button
                        onClick={() => setSelectedItem(item)}
                        className="inline-flex items-center gap-1 text-xs text-zinc-500 hover:text-zinc-200 transition-colors px-2 py-1 hover:bg-zinc-800 rounded"
                      >
                        <FileCode size={12} />
                        YAML
                        <ChevronRight size={11} />
                      </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </div>

      {/* YAML Modal */}
      {selectedItem && (
        <YamlModal item={selectedItem} onClose={() => setSelectedItem(null)} />
      )}
    </div>
  );
}
