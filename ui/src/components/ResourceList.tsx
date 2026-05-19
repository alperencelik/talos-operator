import React, { useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import axios from 'axios';
import {
  FileCode,
  X,
  ChevronRight,
  Trash2,
  Search,
  ArrowUp,
  ArrowDown,
  Check,
  AlertTriangle,
} from 'lucide-react';
import YAML from 'js-yaml';
import { KIND_PATH } from '../App';
import { useEscape } from '../hooks/useEscape';

interface ResourceListProps {
  title: string;
  kind: string;
  items: any[];
  loading: boolean;
  onActionSuccess: (message: string) => void;
  onActionError: (message: string) => void;
  onRefresh: () => void;
}

type StatusTone = 'green' | 'red' | 'yellow' | 'gray';

const TONE_TEXT: Record<StatusTone, string> = {
  green: 'text-green-400',
  red: 'text-red-400',
  yellow: 'text-yellow-400',
  gray: 'text-zinc-500',
};

const TONE_DOT: Record<StatusTone, string> = {
  green: 'bg-green-500',
  red: 'bg-red-500',
  yellow: 'bg-yellow-500',
  gray: 'bg-zinc-600',
};

const IN_PROGRESS_REASONS = new Set([
  'pending',
  'creating',
  'initializing',
  'provisioning',
  'reconciling',
  'inprogress',
  'progressing',
  'updating',
  'upgrading',
]);

interface ResourceStatus {
  label: string;
  tone: StatusTone;
  tooltip?: string;
  rank: number; // for sorting: green=3, yellow=2, red=1, gray=0
}

const TONE_RANK: Record<StatusTone, number> = { green: 3, yellow: 2, red: 1, gray: 0 };

function getStatus(item: any): ResourceStatus {
  const conditions: any[] = item.status?.conditions ?? [];
  if (conditions.length === 0) {
    return { label: '—', tone: 'gray', rank: TONE_RANK.gray };
  }

  const ready = conditions.find((c: any) => c.type === 'Ready');
  if (ready) {
    const tooltip = ready.message || ready.reason;
    if (ready.status === 'True') {
      return { label: 'Ready', tone: 'green', tooltip, rank: TONE_RANK.green };
    }
    if (ready.status === 'Unknown') {
      return { label: ready.reason || 'Unknown', tone: 'yellow', tooltip, rank: TONE_RANK.yellow };
    }
    const reasonLower = (ready.reason ?? '').toLowerCase();
    const tone: StatusTone = IN_PROGRESS_REASONS.has(reasonLower) ? 'yellow' : 'red';
    return { label: ready.reason || 'Not Ready', tone, tooltip, rank: TONE_RANK[tone] };
  }

  const failing = conditions.find((c: any) => c.status === 'False');
  if (failing) {
    return {
      label: 'Degraded',
      tone: 'red',
      tooltip: failing.message || `${failing.type}: ${failing.reason ?? 'False'}`,
      rank: TONE_RANK.red,
    };
  }

  const latest = conditions[conditions.length - 1];
  return {
    label: latest.type ?? '—',
    tone: 'gray',
    tooltip: latest.message || latest.reason,
    rank: TONE_RANK.gray,
  };
}

function getAge(timestamp?: string): { label: string; ms: number } {
  if (!timestamp) return { label: '—', ms: 0 };
  const ms = Date.now() - new Date(timestamp).getTime();
  const minutes = Math.floor(ms / 60000);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);
  let label = 'just now';
  if (days > 0) label = `${days}d`;
  else if (hours > 0) label = `${hours}h`;
  else if (minutes > 0) label = `${minutes}m`;
  return { label, ms };
}

type SortKey = 'name' | 'namespace' | 'status' | 'age';
type SortDir = 'asc' | 'desc';

// ─── YAML edit modal ──────────────────────────────────────────────────────────

interface YamlEditModalProps {
  item: any;
  kind: string;
  onClose: () => void;
  onApplySuccess: () => void;
  onApplyError: (msg: string) => void;
}

function YamlEditModal({ item, kind, onClose, onApplySuccess, onApplyError }: YamlEditModalProps) {
  const original = useMemo(() => YAML.dump(item), [item]);
  const [draft, setDraft] = useState(original);
  const [state, setState] = useState<'idle' | 'applying' | 'success'>('idle');
  useEscape(onClose, state !== 'applying');

  const parseError = useMemo(() => {
    try {
      YAML.load(draft);
      return null;
    } catch (e: any) {
      return e?.message ?? 'Invalid YAML';
    }
  }, [draft]);

  const dirty = draft !== original;
  const canApply = dirty && !parseError && state !== 'applying';

  const handleApply = async () => {
    setState('applying');
    try {
      await axios.post('/api/apply', draft, { headers: { 'Content-Type': 'application/x-yaml' } });
      setState('success');
      onApplySuccess();
      setTimeout(onClose, 600);
    } catch (err) {
      setState('idle');
      let msg = 'Apply failed';
      if (axios.isAxiosError(err)) {
        if (err.response) {
          const data =
            typeof err.response.data === 'string'
              ? err.response.data
              : JSON.stringify(err.response.data);
          msg = `Apply failed (${err.response.status}): ${data}`;
        } else if (err.request) {
          msg = 'Apply failed: no response from server';
        } else {
          msg = `Apply failed: ${err.message}`;
        }
      }
      onApplyError(msg);
    }
  };

  return (
    <div
      className="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center z-50 p-4"
      onClick={onClose}
      role="dialog"
      aria-modal="true"
      aria-label={`Edit ${kind} ${item?.metadata?.name ?? ''}`}
    >
      <div
        className="bg-zinc-900 border border-zinc-700 rounded-xl shadow-2xl w-full max-w-3xl max-h-[90vh] flex flex-col"
        onClick={e => e.stopPropagation()}
      >
        <div className="flex items-center justify-between px-5 py-4 border-b border-zinc-800 flex-shrink-0">
          <div className="flex items-center gap-2.5 min-w-0">
            <FileCode size={15} className="text-zinc-500 flex-shrink-0" />
            <span className="text-sm font-semibold text-zinc-100 truncate">
              {item?.metadata?.name}
            </span>
            <span className="text-xs text-zinc-500 flex-shrink-0">
              {item?.metadata?.namespace}
            </span>
            {dirty && (
              <span className="text-[10px] text-yellow-400 bg-yellow-950 border border-yellow-900 px-1.5 py-0.5 rounded">
                modified
              </span>
            )}
          </div>
          <button
            onClick={onClose}
            className="text-zinc-500 hover:text-zinc-300 transition-colors flex-shrink-0 ml-3 p-1 hover:bg-zinc-800 rounded"
            aria-label="Close"
          >
            <X size={15} />
          </button>
        </div>

        <textarea
          value={draft}
          onChange={e => setDraft(e.target.value)}
          spellCheck={false}
          className="flex-1 bg-zinc-950 text-zinc-200 text-xs font-mono leading-relaxed p-5 resize-none focus:outline-none"
        />

        <div className="flex items-center justify-between gap-3 px-5 py-3 border-t border-zinc-800 flex-shrink-0">
          <div className="text-xs text-zinc-500 min-w-0 truncate" title={parseError ?? undefined}>
            {parseError ? (
              <span className="text-red-400 flex items-center gap-1.5">
                <AlertTriangle size={12} /> {parseError}
              </span>
            ) : dirty ? (
              'Ready to apply'
            ) : (
              'No changes'
            )}
          </div>
          <div className="flex items-center gap-2 flex-shrink-0">
            <button
              onClick={onClose}
              className="px-3 py-1.5 rounded-md text-xs font-medium bg-zinc-800 text-zinc-300 hover:bg-zinc-700 hover:text-zinc-100 transition-colors border border-zinc-700"
            >
              Cancel
            </button>
            <button
              onClick={handleApply}
              disabled={!canApply}
              className={`flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-all disabled:opacity-40 disabled:cursor-not-allowed ${
                state === 'success'
                  ? 'bg-green-900 text-green-300 border border-green-800'
                  : 'bg-brand text-white hover:bg-brand-hover'
              }`}
            >
              {state === 'applying' ? (
                <>
                  <div className="w-3 h-3 border border-white/30 border-t-white rounded-full animate-spin" />
                  Applying…
                </>
              ) : state === 'success' ? (
                <>
                  <Check size={12} />
                  Applied
                </>
              ) : (
                'Apply'
              )}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

// ─── Delete confirmation ──────────────────────────────────────────────────────

interface DeleteModalProps {
  item: any;
  kind: string;
  onClose: () => void;
  onDeleted: () => void;
  onError: (msg: string) => void;
}

function DeleteModal({ item, kind, onClose, onDeleted, onError }: DeleteModalProps) {
  const [state, setState] = useState<'idle' | 'deleting'>('idle');
  useEscape(onClose, state !== 'deleting');

  const name = item?.metadata?.name ?? '';
  const namespace = item?.metadata?.namespace ?? '';

  const handleDelete = async () => {
    setState('deleting');
    try {
      await axios.delete(`/api/resources/${kind}/${namespace}/${name}`);
      onDeleted();
    } catch (err) {
      setState('idle');
      let msg = 'Delete failed';
      if (axios.isAxiosError(err)) {
        if (err.response) {
          const data =
            typeof err.response.data === 'string'
              ? err.response.data
              : JSON.stringify(err.response.data);
          msg = `Delete failed (${err.response.status}): ${data}`;
        } else if (err.request) {
          msg = 'Delete failed: no response from server';
        } else {
          msg = `Delete failed: ${err.message}`;
        }
      }
      onError(msg);
    }
  };

  return (
    <div
      className="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center z-50 p-4"
      onClick={onClose}
      role="dialog"
      aria-modal="true"
    >
      <div
        className="bg-zinc-900 border border-zinc-700 rounded-xl shadow-2xl w-full max-w-md"
        onClick={e => e.stopPropagation()}
      >
        <div className="px-5 py-4 border-b border-zinc-800 flex items-center gap-2.5">
          <div className="w-8 h-8 bg-red-950 border border-red-900 rounded-lg flex items-center justify-center flex-shrink-0">
            <Trash2 size={14} className="text-red-400" />
          </div>
          <h2 className="text-sm font-semibold text-zinc-100">Delete {kind}?</h2>
        </div>
        <div className="px-5 py-4 text-sm text-zinc-400">
          This will request deletion of{' '}
          <code className="text-zinc-200 bg-zinc-800 px-1.5 py-0.5 rounded text-xs">
            {namespace}/{name}
          </code>
          . Any owned child resources may be removed by the operator.
        </div>
        <div className="px-5 py-3 border-t border-zinc-800 flex items-center justify-end gap-2">
          <button
            onClick={onClose}
            disabled={state === 'deleting'}
            className="px-3 py-1.5 rounded-md text-xs font-medium bg-zinc-800 text-zinc-300 hover:bg-zinc-700 hover:text-zinc-100 transition-colors border border-zinc-700 disabled:opacity-40"
          >
            Cancel
          </button>
          <button
            onClick={handleDelete}
            disabled={state === 'deleting'}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium bg-red-600 text-white hover:bg-red-500 transition-colors disabled:opacity-50"
          >
            {state === 'deleting' ? (
              <>
                <div className="w-3 h-3 border border-white/30 border-t-white rounded-full animate-spin" />
                Deleting…
              </>
            ) : (
              <>
                <Trash2 size={12} />
                Delete
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
}

// ─── Sort header ──────────────────────────────────────────────────────────────

interface SortHeaderProps {
  label: string;
  field: SortKey;
  sortKey: SortKey;
  sortDir: SortDir;
  align?: 'left' | 'right';
  onClick: (field: SortKey) => void;
}

function SortHeader({ label, field, sortKey, sortDir, align = 'left', onClick }: SortHeaderProps) {
  const active = sortKey === field;
  return (
    <th className={`px-5 py-3 text-${align} text-xs font-medium text-zinc-500 uppercase tracking-wider`}>
      <button
        onClick={() => onClick(field)}
        className={`inline-flex items-center gap-1 hover:text-zinc-300 transition-colors ${active ? 'text-zinc-300' : ''}`}
      >
        {label}
        {active && (sortDir === 'asc' ? <ArrowUp size={11} /> : <ArrowDown size={11} />)}
      </button>
    </th>
  );
}

// ─── Main component ───────────────────────────────────────────────────────────

export default function ResourceList({
  title,
  kind,
  items,
  loading,
  onActionSuccess,
  onActionError,
  onRefresh,
}: ResourceListProps) {
  const [editItem, setEditItem] = useState<any>(null);
  const [deleteItem, setDeleteItem] = useState<any>(null);
  const [query, setQuery] = useState('');
  const [namespace, setNamespace] = useState<string>('__all__');
  const [sortKey, setSortKey] = useState<SortKey>('name');
  const [sortDir, setSortDir] = useState<SortDir>('asc');

  const namespaces = useMemo(() => {
    const set = new Set<string>();
    items.forEach(i => i.metadata?.namespace && set.add(i.metadata.namespace));
    return Array.from(set).sort();
  }, [items]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return items.filter(i => {
      if (namespace !== '__all__' && i.metadata?.namespace !== namespace) return false;
      if (!q) return true;
      const name = i.metadata?.name?.toLowerCase() ?? '';
      const ns = i.metadata?.namespace?.toLowerCase() ?? '';
      return name.includes(q) || ns.includes(q);
    });
  }, [items, query, namespace]);

  const sorted = useMemo(() => {
    const copy = filtered.slice();
    copy.sort((a, b) => {
      let cmp = 0;
      switch (sortKey) {
        case 'name':
          cmp = (a.metadata?.name ?? '').localeCompare(b.metadata?.name ?? '');
          break;
        case 'namespace':
          cmp = (a.metadata?.namespace ?? '').localeCompare(b.metadata?.namespace ?? '');
          break;
        case 'status':
          cmp = getStatus(a).rank - getStatus(b).rank;
          break;
        case 'age':
          // older = larger ms; asc means oldest first
          cmp = getAge(a.metadata?.creationTimestamp).ms - getAge(b.metadata?.creationTimestamp).ms;
          break;
      }
      return sortDir === 'asc' ? cmp : -cmp;
    });
    return copy;
  }, [filtered, sortKey, sortDir]);

  const toggleSort = (field: SortKey) => {
    if (field === sortKey) {
      setSortDir(d => (d === 'asc' ? 'desc' : 'asc'));
    } else {
      setSortKey(field);
      setSortDir('asc');
    }
  };

  return (
    <div className="p-6">
      {/* Page header */}
      <div className="mb-5 flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-zinc-100">{title}</h1>
          <p className="text-sm text-zinc-500 mt-0.5">{kind} resources in the cluster</p>
        </div>
        {!loading && (
          <span className="text-xs text-zinc-600 bg-zinc-900 border border-zinc-800 px-2.5 py-1 rounded-full">
            {filtered.length === items.length
              ? `${items.length} ${items.length === 1 ? 'resource' : 'resources'}`
              : `${filtered.length} of ${items.length}`}
          </span>
        )}
      </div>

      {/* Toolbar */}
      <div className="mb-4 flex flex-col sm:flex-row gap-2">
        <div className="relative flex-1 min-w-0">
          <Search
            size={13}
            className="absolute left-2.5 top-1/2 -translate-y-1/2 text-zinc-600 pointer-events-none"
          />
          <input
            value={query}
            onChange={e => setQuery(e.target.value)}
            placeholder="Search by name or namespace…"
            className="w-full bg-zinc-900 border border-zinc-800 rounded-md pl-8 pr-3 py-1.5 text-sm text-zinc-200 placeholder:text-zinc-600 focus:outline-none focus:border-zinc-700 transition-colors"
          />
        </div>
        <select
          value={namespace}
          onChange={e => setNamespace(e.target.value)}
          className="bg-zinc-900 border border-zinc-800 rounded-md px-3 py-1.5 text-sm text-zinc-200 focus:outline-none focus:border-zinc-700 transition-colors cursor-pointer"
        >
          <option value="__all__">All namespaces</option>
          {namespaces.map(ns => (
            <option key={ns} value={ns}>
              {ns}
            </option>
          ))}
        </select>
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
        ) : sorted.length === 0 ? (
          <div className="text-center py-16 px-8">
            <div className="text-zinc-600 text-sm">No matches</div>
            <div className="text-zinc-700 text-xs mt-1">Try a different search or namespace.</div>
          </div>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-zinc-800">
                <SortHeader label="Name" field="name" sortKey={sortKey} sortDir={sortDir} onClick={toggleSort} />
                <SortHeader label="Namespace" field="namespace" sortKey={sortKey} sortDir={sortDir} onClick={toggleSort} />
                <SortHeader label="Status" field="status" sortKey={sortKey} sortDir={sortDir} onClick={toggleSort} />
                <SortHeader label="Age" field="age" sortKey={sortKey} sortDir={sortDir} onClick={toggleSort} />
                <th className="px-5 py-3 text-right text-xs font-medium text-zinc-500 uppercase tracking-wider">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-800">
              {sorted.map((item, i) => {
                const status = getStatus(item);
                const age = getAge(item.metadata?.creationTimestamp);
                return (
                  <tr key={`${item.metadata?.namespace}/${item.metadata?.name}/${i}`} className="hover:bg-zinc-800/30 transition-colors">
                    <td className="px-5 py-3.5">
                      {item.metadata?.name && KIND_PATH[kind] ? (
                        <Link
                          to={`${KIND_PATH[kind]}/${item.metadata?.namespace}/${item.metadata?.name}`}
                          className="font-medium text-zinc-200 hover:text-brand transition-colors"
                        >
                          {item.metadata.name}
                        </Link>
                      ) : (
                        <span className="font-medium text-zinc-200">{item.metadata?.name ?? '—'}</span>
                      )}
                    </td>
                    <td className="px-5 py-3.5">
                      <span className="text-zinc-500">{item.metadata?.namespace ?? '—'}</span>
                    </td>
                    <td className="px-5 py-3.5">
                      <div className="flex items-center gap-2" title={status.tooltip}>
                        <span className={`w-1.5 h-1.5 rounded-full flex-shrink-0 ${TONE_DOT[status.tone]}`} />
                        <span className={`text-xs ${TONE_TEXT[status.tone]}`}>{status.label}</span>
                      </div>
                    </td>
                    <td className="px-5 py-3.5">
                      <span className="text-zinc-500 text-xs">{age.label}</span>
                    </td>
                    <td className="px-5 py-3.5 text-right">
                      <div className="inline-flex items-center gap-1">
                        <button
                          onClick={() => setEditItem(item)}
                          className="inline-flex items-center gap-1 text-xs text-zinc-500 hover:text-zinc-200 transition-colors px-2 py-1 hover:bg-zinc-800 rounded"
                        >
                          <FileCode size={12} />
                          YAML
                          <ChevronRight size={11} />
                        </button>
                        <button
                          onClick={() => setDeleteItem(item)}
                          className="inline-flex items-center gap-1 text-xs text-zinc-500 hover:text-red-400 transition-colors px-2 py-1 hover:bg-zinc-800 rounded"
                          aria-label={`Delete ${item.metadata?.name}`}
                        >
                          <Trash2 size={12} />
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </div>

      {editItem && (
        <YamlEditModal
          item={editItem}
          kind={kind}
          onClose={() => setEditItem(null)}
          onApplySuccess={() => {
            onActionSuccess(`${kind} ${editItem.metadata?.name} updated`);
            onRefresh();
          }}
          onApplyError={onActionError}
        />
      )}

      {deleteItem && (
        <DeleteModal
          item={deleteItem}
          kind={kind}
          onClose={() => setDeleteItem(null)}
          onDeleted={() => {
            const name = deleteItem.metadata?.name;
            setDeleteItem(null);
            onActionSuccess(`${kind} ${name} deleted`);
            onRefresh();
          }}
          onError={onActionError}
        />
      )}
    </div>
  );
}
