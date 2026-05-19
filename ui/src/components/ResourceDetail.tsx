import React, { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import axios from 'axios';
import YAML from 'js-yaml';
import {
  ArrowLeft,
  Trash2,
  Check,
  AlertTriangle,
  RefreshCw,
} from 'lucide-react';
import { KIND_PATH, queryKeys, Resources } from '../App';
import { useEscape } from '../hooks/useEscape';

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
const TONE_BG: Record<StatusTone, string> = {
  green: 'bg-green-950 border-green-900',
  red: 'bg-red-950 border-red-900',
  yellow: 'bg-yellow-950 border-yellow-900',
  gray: 'bg-zinc-900 border-zinc-800',
};

const IN_PROGRESS_REASONS = new Set([
  'pending', 'creating', 'initializing', 'provisioning',
  'reconciling', 'inprogress', 'progressing', 'updating', 'upgrading',
]);

interface ResourceStatus {
  label: string;
  tone: StatusTone;
  tooltip?: string;
}

function getStatus(item: any): ResourceStatus {
  const conditions: any[] = item?.status?.conditions ?? [];
  if (conditions.length === 0) return { label: '—', tone: 'gray' };
  const ready = conditions.find((c: any) => c.type === 'Ready');
  if (ready) {
    const tooltip = ready.message || ready.reason;
    if (ready.status === 'True') return { label: 'Ready', tone: 'green', tooltip };
    if (ready.status === 'Unknown') return { label: ready.reason || 'Unknown', tone: 'yellow', tooltip };
    const reasonLower = (ready.reason ?? '').toLowerCase();
    const tone: StatusTone = IN_PROGRESS_REASONS.has(reasonLower) ? 'yellow' : 'red';
    return { label: ready.reason || 'Not Ready', tone, tooltip };
  }
  const failing = conditions.find((c: any) => c.status === 'False');
  if (failing) {
    return {
      label: 'Degraded',
      tone: 'red',
      tooltip: failing.message || `${failing.type}: ${failing.reason ?? 'False'}`,
    };
  }
  return { label: conditions[conditions.length - 1].type ?? '—', tone: 'gray' };
}

function formatAge(timestamp?: string): string {
  if (!timestamp) return '—';
  const ms = Date.now() - new Date(timestamp).getTime();
  const m = Math.floor(ms / 60000);
  const h = Math.floor(m / 60);
  const d = Math.floor(h / 24);
  if (d > 0) return `${d}d`;
  if (h > 0) return `${h}h`;
  if (m > 0) return `${m}m`;
  return 'just now';
}

const RESOURCE_KEYS: { kind: string; key: keyof Resources }[] = [
  { kind: 'TalosCluster', key: 'talosClusters' },
  { kind: 'TalosControlPlane', key: 'talosControlPlanes' },
  { kind: 'TalosWorker', key: 'talosWorkers' },
  { kind: 'TalosMachine', key: 'talosMachines' },
  { kind: 'TalosClusterAddon', key: 'talosClusterAddons' },
  { kind: 'TalosClusterAddonRelease', key: 'talosClusterAddonReleases' },
  { kind: 'TalosEtcdBackup', key: 'talosEtcdBackups' },
  { kind: 'TalosEtcdBackupSchedule', key: 'talosEtcdBackupSchedules' },
];

function findOwnedChildren(resources: Resources | null, parentKind: string, parentName: string) {
  if (!resources) return [];
  const owned: { kind: string; item: any }[] = [];
  for (const { kind, key } of RESOURCE_KEYS) {
    const arr = (resources[key] ?? []) as any[];
    for (const item of arr) {
      const refs = item.metadata?.ownerReferences ?? [];
      if (refs.some((r: any) => r.kind === parentKind && r.name === parentName)) {
        owned.push({ kind, item });
      }
    }
  }
  return owned;
}

// ─── Component ────────────────────────────────────────────────────────────────

interface ResourceDetailProps {
  kind: string;
  resources: Resources | null;
  loading: boolean;
  onActionSuccess: (msg: string) => void;
  onActionError: (msg: string) => void;
  onRefresh: () => void;
}

type Tab = 'summary' | 'conditions' | 'yaml' | 'events' | 'children';

export default function ResourceDetail({
  kind,
  resources,
  loading,
  onActionSuccess,
  onActionError,
  onRefresh,
}: ResourceDetailProps) {
  const { namespace, name } = useParams<{ namespace: string; name: string }>();
  const navigate = useNavigate();
  const qc = useQueryClient();
  const [tab, setTab] = useState<Tab>('summary');

  // Look the item up in the cached list first; only hit the network if we
  // arrived via a deep link and the list isn't loaded yet.
  const cachedItem = useMemo(() => {
    if (!resources || !namespace || !name) return null;
    const entry = RESOURCE_KEYS.find(e => e.kind === kind);
    if (!entry) return null;
    const arr = (resources[entry.key] ?? []) as any[];
    return (
      arr.find(
        i => i.metadata?.name === name && i.metadata?.namespace === namespace
      ) ?? null
    );
  }, [resources, kind, namespace, name]);

  const singleQuery = useQuery({
    queryKey: queryKeys.resource(kind, namespace ?? '', name ?? ''),
    queryFn: async () => {
      const res = await axios.get(`/api/resources/${kind}/${namespace}/${name}`);
      return res.data;
    },
    enabled: !cachedItem && !!namespace && !!name && !loading,
    refetchInterval: 10000,
    retry: false,
  });

  const fetched = singleQuery.data ?? null;
  const fetching = singleQuery.isLoading || singleQuery.isFetching;
  const fetchError = singleQuery.isError
    ? axios.isAxiosError(singleQuery.error) && singleQuery.error.response
      ? `${singleQuery.error.response.status}: ${
          typeof singleQuery.error.response.data === 'string'
            ? singleQuery.error.response.data
            : 'not found'
        }`
      : 'Failed to fetch resource'
    : null;

  const item = cachedItem ?? fetched;
  const owned = useMemo(() => findOwnedChildren(resources, kind, name ?? ''), [resources, kind, name]);

  if (loading && !item) {
    return (
      <div className="p-6 flex items-center gap-3 text-sm text-zinc-500">
        <div className="w-4 h-4 border-2 border-zinc-700 border-t-brand rounded-full animate-spin" />
        Loading {kind}…
      </div>
    );
  }

  if (!item) {
    return (
      <div className="p-6">
        <Link to={KIND_PATH[kind]} className="inline-flex items-center gap-1 text-sm text-zinc-500 hover:text-zinc-300">
          <ArrowLeft size={14} /> Back
        </Link>
        <div className="mt-6 bg-zinc-900 border border-zinc-800 rounded-xl p-8 text-center">
          {fetching ? (
            <div className="text-zinc-500 text-sm">Looking up {namespace}/{name}…</div>
          ) : (
            <>
              <div className="text-zinc-300 text-sm font-medium">Resource not found</div>
              <div className="text-zinc-600 text-xs mt-1">
                {fetchError ?? `${kind} ${namespace}/${name} doesn't exist in the cluster.`}
              </div>
            </>
          )}
        </div>
      </div>
    );
  }

  const status = getStatus(item);
  const conditions: any[] = item.status?.conditions ?? [];
  const tabs: { id: Tab; label: string; count?: number }[] = [
    { id: 'summary', label: 'Summary' },
    { id: 'conditions', label: 'Conditions', count: conditions.length },
    { id: 'yaml', label: 'YAML' },
    { id: 'events', label: 'Events' },
    { id: 'children', label: 'Children', count: owned.length },
  ];

  return (
    <div className="p-6 max-w-5xl">
      {/* Breadcrumb + title */}
      <Link
        to={KIND_PATH[kind]}
        className="inline-flex items-center gap-1 text-xs text-zinc-500 hover:text-zinc-300 transition-colors mb-3"
      >
        <ArrowLeft size={12} /> {kind.replace(/^Talos/, '')}s
      </Link>
      <div className="flex items-start justify-between mb-6 gap-4">
        <div className="min-w-0">
          <h1 className="text-xl font-semibold text-zinc-100 truncate">{item.metadata?.name}</h1>
          <div className="flex items-center gap-3 mt-1 text-sm">
            <span className="text-zinc-500">{kind}</span>
            <span className="text-zinc-700">·</span>
            <span className="text-zinc-500">{item.metadata?.namespace}</span>
            <span className="text-zinc-700">·</span>
            <span className="text-zinc-500">{formatAge(item.metadata?.creationTimestamp)} old</span>
            <span className="text-zinc-700">·</span>
            <div className="flex items-center gap-1.5" title={status.tooltip}>
              <span className={`w-1.5 h-1.5 rounded-full ${TONE_DOT[status.tone]}`} />
              <span className={TONE_TEXT[status.tone]}>{status.label}</span>
            </div>
          </div>
        </div>
        <DetailActions
          kind={kind}
          item={item}
          onActionSuccess={msg => {
            onActionSuccess(msg);
            onRefresh();
          }}
          onActionError={onActionError}
          onDeleted={() => navigate(KIND_PATH[kind])}
        />
      </div>

      {/* Tabs */}
      <div className="border-b border-zinc-800 mb-5 flex items-center gap-1">
        {tabs.map(t => {
          const active = tab === t.id;
          return (
            <button
              key={t.id}
              onClick={() => setTab(t.id)}
              className={`px-3 py-2 text-sm transition-colors -mb-px border-b-2 ${
                active
                  ? 'text-zinc-100 border-brand'
                  : 'text-zinc-500 border-transparent hover:text-zinc-300'
              }`}
            >
              {t.label}
              {typeof t.count === 'number' && (
                <span className="ml-1.5 text-[10px] text-zinc-600">{t.count}</span>
              )}
            </button>
          );
        })}
      </div>

      {/* Tab content */}
      {tab === 'summary' && <SummaryTab item={item} status={status} />}
      {tab === 'conditions' && <ConditionsTab conditions={conditions} />}
      {tab === 'yaml' && (
        <YamlTab
          item={item}
          kind={kind}
          onActionSuccess={msg => {
            onActionSuccess(msg);
            onRefresh();
            // Make sure the single-resource query picks up the fresh server
            // state immediately, not on the next 10s tick.
            qc.invalidateQueries({ queryKey: queryKeys.resource(kind, namespace ?? '', name ?? '') });
          }}
          onActionError={onActionError}
        />
      )}
      {tab === 'events' && <EventsTab kind={kind} namespace={namespace!} name={name!} />}
      {tab === 'children' && <ChildrenTab owned={owned} />}
    </div>
  );
}

// ─── Actions (Delete) ─────────────────────────────────────────────────────────

function DetailActions({
  kind,
  item,
  onActionSuccess,
  onActionError,
  onDeleted,
}: {
  kind: string;
  item: any;
  onActionSuccess: (msg: string) => void;
  onActionError: (msg: string) => void;
  onDeleted: () => void;
}) {
  const [confirming, setConfirming] = useState(false);
  const [deleting, setDeleting] = useState(false);
  useEscape(() => setConfirming(false), confirming && !deleting);

  const namespace = item.metadata?.namespace;
  const name = item.metadata?.name;

  const handleDelete = async () => {
    setDeleting(true);
    try {
      await axios.delete(`/api/resources/${kind}/${namespace}/${name}`);
      onActionSuccess(`${kind} ${name} deleted`);
      onDeleted();
    } catch (err) {
      setDeleting(false);
      setConfirming(false);
      let msg = 'Delete failed';
      if (axios.isAxiosError(err) && err.response) {
        msg = `Delete failed (${err.response.status}): ${
          typeof err.response.data === 'string' ? err.response.data : ''
        }`;
      }
      onActionError(msg);
    }
  };

  return (
    <>
      <button
        onClick={() => setConfirming(true)}
        className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium bg-zinc-900 border border-zinc-800 text-zinc-400 hover:text-red-400 hover:border-red-900 transition-colors flex-shrink-0"
      >
        <Trash2 size={12} />
        Delete
      </button>

      {confirming && (
        <div
          className="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center z-50 p-4"
          onClick={() => !deleting && setConfirming(false)}
          role="dialog"
          aria-modal="true"
        >
          <div
            className="bg-zinc-900 border border-zinc-700 rounded-xl shadow-2xl w-full max-w-md"
            onClick={e => e.stopPropagation()}
          >
            <div className="px-5 py-4 border-b border-zinc-800 flex items-center gap-2.5">
              <div className="w-8 h-8 bg-red-950 border border-red-900 rounded-lg flex items-center justify-center">
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
                onClick={() => setConfirming(false)}
                disabled={deleting}
                className="px-3 py-1.5 rounded-md text-xs font-medium bg-zinc-800 text-zinc-300 hover:bg-zinc-700 transition-colors border border-zinc-700 disabled:opacity-40"
              >
                Cancel
              </button>
              <button
                onClick={handleDelete}
                disabled={deleting}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium bg-red-600 text-white hover:bg-red-500 transition-colors disabled:opacity-50"
              >
                {deleting ? (
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
      )}
    </>
  );
}

// ─── Summary tab ──────────────────────────────────────────────────────────────

function SummaryTab({ item, status }: { item: any; status: ResourceStatus }) {
  const meta = item.metadata ?? {};
  const spec = item.spec ?? {};

  // Heuristic: surface common fields if present.
  const summaryEntries: { label: string; value: React.ReactNode }[] = [];
  if (spec.version) summaryEntries.push({ label: 'Talos Version', value: spec.version });
  if (spec.kubeVersion) summaryEntries.push({ label: 'Kubernetes Version', value: spec.kubeVersion });
  if (spec.mode) summaryEntries.push({ label: 'Mode', value: spec.mode });
  if (spec.endpoint) summaryEntries.push({ label: 'Endpoint', value: <code className="text-xs">{spec.endpoint}</code> });
  if (typeof spec.replicas === 'number') summaryEntries.push({ label: 'Replicas', value: spec.replicas });
  if (Array.isArray(spec.metalSpec?.machines)) {
    summaryEntries.push({ label: 'Machines', value: `${spec.metalSpec.machines.length} configured` });
  }
  if (spec.controlPlaneRef?.name) {
    summaryEntries.push({
      label: 'Control Plane Ref',
      value: <code className="text-xs">{spec.controlPlaneRef.name}</code>,
    });
  }
  if (spec.clusterRef?.name) {
    summaryEntries.push({ label: 'Cluster Ref', value: <code className="text-xs">{spec.clusterRef.name}</code> });
  }
  if (spec.schedule) summaryEntries.push({ label: 'Schedule', value: <code className="text-xs">{spec.schedule}</code> });

  return (
    <div className="space-y-4">
      <div className={`rounded-xl border p-4 ${TONE_BG[status.tone]}`}>
        <div className="flex items-center gap-2">
          <span className={`w-2 h-2 rounded-full ${TONE_DOT[status.tone]}`} />
          <span className={`text-sm font-medium ${TONE_TEXT[status.tone]}`}>{status.label}</span>
        </div>
        {status.tooltip && <p className="text-xs text-zinc-400 mt-1.5">{status.tooltip}</p>}
      </div>

      <div className="grid sm:grid-cols-2 gap-3">
        <SummaryCard title="Metadata">
          <KV label="Name" value={meta.name} />
          <KV label="Namespace" value={meta.namespace} />
          <KV label="Age" value={formatAge(meta.creationTimestamp)} />
          <KV label="UID" value={<code className="text-[10px] break-all">{meta.uid ?? '—'}</code>} />
          {meta.ownerReferences?.[0] && (
            <KV
              label="Owner"
              value={`${meta.ownerReferences[0].kind} / ${meta.ownerReferences[0].name}`}
            />
          )}
        </SummaryCard>

        {summaryEntries.length > 0 && (
          <SummaryCard title="Spec">
            {summaryEntries.map(e => (
              <KV key={e.label} label={e.label} value={e.value} />
            ))}
          </SummaryCard>
        )}
      </div>

      {meta.labels && Object.keys(meta.labels).length > 0 && (
        <SummaryCard title="Labels">
          <div className="flex flex-wrap gap-1.5">
            {Object.entries(meta.labels).map(([k, v]) => (
              <span key={k} className="text-[11px] text-zinc-300 bg-zinc-800 px-2 py-0.5 rounded font-mono">
                {k}={String(v)}
              </span>
            ))}
          </div>
        </SummaryCard>
      )}
    </div>
  );
}

function SummaryCard({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4">
      <div className="text-xs font-semibold text-zinc-500 uppercase tracking-wider mb-3">{title}</div>
      <div className="space-y-2">{children}</div>
    </div>
  );
}

function KV({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex items-start justify-between gap-3 text-sm">
      <span className="text-zinc-500 text-xs flex-shrink-0">{label}</span>
      <span className="text-zinc-200 text-right min-w-0 truncate">{value ?? '—'}</span>
    </div>
  );
}

// ─── Conditions tab ───────────────────────────────────────────────────────────

function ConditionsTab({ conditions }: { conditions: any[] }) {
  if (conditions.length === 0) {
    return (
      <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-8 text-center text-sm text-zinc-500">
        No conditions reported yet.
      </div>
    );
  }
  // Newest first
  const sorted = conditions
    .slice()
    .sort((a, b) =>
      new Date(b.lastTransitionTime ?? 0).getTime() - new Date(a.lastTransitionTime ?? 0).getTime()
    );

  return (
    <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-zinc-800">
            <th className="px-4 py-2.5 text-left text-xs font-medium text-zinc-500 uppercase tracking-wider">Type</th>
            <th className="px-4 py-2.5 text-left text-xs font-medium text-zinc-500 uppercase tracking-wider">Status</th>
            <th className="px-4 py-2.5 text-left text-xs font-medium text-zinc-500 uppercase tracking-wider">Reason</th>
            <th className="px-4 py-2.5 text-left text-xs font-medium text-zinc-500 uppercase tracking-wider">Message</th>
            <th className="px-4 py-2.5 text-left text-xs font-medium text-zinc-500 uppercase tracking-wider">Last Change</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-zinc-800">
          {sorted.map((c, i) => {
            const tone: StatusTone =
              c.status === 'True' && c.type === 'Ready'
                ? 'green'
                : c.status === 'True'
                ? 'green'
                : c.status === 'Unknown'
                ? 'yellow'
                : 'red';
            return (
              <tr key={i} className="hover:bg-zinc-800/30">
                <td className="px-4 py-3 text-zinc-200 font-medium">{c.type}</td>
                <td className="px-4 py-3">
                  <div className="flex items-center gap-1.5">
                    <span className={`w-1.5 h-1.5 rounded-full ${TONE_DOT[tone]}`} />
                    <span className={`text-xs ${TONE_TEXT[tone]}`}>{c.status}</span>
                  </div>
                </td>
                <td className="px-4 py-3 text-zinc-400 text-xs">{c.reason ?? '—'}</td>
                <td className="px-4 py-3 text-zinc-400 text-xs">{c.message ?? '—'}</td>
                <td className="px-4 py-3 text-zinc-500 text-xs whitespace-nowrap">
                  {formatAge(c.lastTransitionTime)} ago
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

// ─── YAML tab ─────────────────────────────────────────────────────────────────

function YamlTab({
  item,
  kind,
  onActionSuccess,
  onActionError,
}: {
  item: any;
  kind: string;
  onActionSuccess: (msg: string) => void;
  onActionError: (msg: string) => void;
}) {
  const original = useMemo(() => YAML.dump(item), [item]);
  const [draft, setDraft] = useState(original);
  const [state, setState] = useState<'idle' | 'applying' | 'success'>('idle');

  useEffect(() => {
    setDraft(original);
  }, [original]);

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
      onActionSuccess(`${kind} ${item.metadata?.name} updated`);
      setTimeout(() => setState('idle'), 1500);
    } catch (err) {
      setState('idle');
      let msg = 'Apply failed';
      if (axios.isAxiosError(err) && err.response) {
        msg = `Apply failed (${err.response.status}): ${
          typeof err.response.data === 'string' ? err.response.data : ''
        }`;
      }
      onActionError(msg);
    }
  };

  return (
    <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden flex flex-col">
      <div className="flex items-center justify-between px-4 py-2.5 border-b border-zinc-800 text-xs">
        <div className="text-zinc-500" title={parseError ?? undefined}>
          {parseError ? (
            <span className="text-red-400 flex items-center gap-1.5">
              <AlertTriangle size={12} /> {parseError}
            </span>
          ) : dirty ? (
            <span className="text-yellow-400">Modified</span>
          ) : (
            'No changes'
          )}
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setDraft(original)}
            disabled={!dirty}
            className="px-2.5 py-1 rounded text-xs font-medium bg-zinc-800 text-zinc-300 hover:bg-zinc-700 transition-colors border border-zinc-700 disabled:opacity-40 disabled:cursor-not-allowed"
          >
            Discard
          </button>
          <button
            onClick={handleApply}
            disabled={!canApply}
            className={`flex items-center gap-1.5 px-2.5 py-1 rounded text-xs font-medium transition-all disabled:opacity-40 disabled:cursor-not-allowed ${
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
      <textarea
        value={draft}
        onChange={e => setDraft(e.target.value)}
        spellCheck={false}
        className="bg-zinc-950 text-zinc-200 text-xs font-mono leading-relaxed p-4 min-h-[60vh] resize-none focus:outline-none"
      />
    </div>
  );
}

// ─── Events tab ───────────────────────────────────────────────────────────────

function EventsTab({ kind, namespace, name }: { kind: string; namespace: string; name: string }) {
  const { data, isLoading, isError, error, refetch } = useQuery<any[]>({
    queryKey: queryKeys.events(kind, namespace, name),
    queryFn: async () => {
      const res = await axios.get(`/api/events/${kind}/${namespace}/${name}`);
      return res.data ?? [];
    },
    refetchInterval: 10000,
  });
  const events = data ?? null;
  const loading = isLoading;
  const errorLabel = isError
    ? axios.isAxiosError(error) && error.response
      ? `${error.response.status}`
      : 'fetch failed'
    : null;
  const fetchEvents = () => refetch();

  if (loading) {
    return (
      <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-8 text-center text-sm text-zinc-500">
        Loading events…
      </div>
    );
  }
  if (errorLabel) {
    return (
      <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6 text-sm text-red-400">
        Failed to load events: {errorLabel}
      </div>
    );
  }
  if (!events || events.length === 0) {
    return (
      <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-8 text-center text-sm text-zinc-500">
        No events recorded for this resource.
      </div>
    );
  }

  // Sort newest first by lastTimestamp || eventTime
  const sorted = events.slice().sort((a, b) => {
    const ta = new Date(a.lastTimestamp ?? a.eventTime ?? 0).getTime();
    const tb = new Date(b.lastTimestamp ?? b.eventTime ?? 0).getTime();
    return tb - ta;
  });

  return (
    <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden">
      <div className="flex items-center justify-between px-4 py-2.5 border-b border-zinc-800">
        <span className="text-xs text-zinc-500">{sorted.length} events</span>
        <button
          onClick={fetchEvents}
          className="flex items-center gap-1.5 text-xs text-zinc-500 hover:text-zinc-300 transition-colors px-2 py-1 rounded hover:bg-zinc-800"
        >
          <RefreshCw size={11} />
          Reload
        </button>
      </div>
      <div className="divide-y divide-zinc-800">
        {sorted.map((e, i) => {
          const tone: StatusTone = e.type === 'Warning' ? 'yellow' : 'gray';
          return (
            <div key={i} className="px-4 py-3 hover:bg-zinc-800/30">
              <div className="flex items-center gap-2 text-xs">
                <span className={`w-1.5 h-1.5 rounded-full ${TONE_DOT[tone]}`} />
                <span className={`font-medium ${TONE_TEXT[tone]}`}>{e.type}</span>
                <span className="text-zinc-400 font-medium">{e.reason}</span>
                <span className="text-zinc-600 ml-auto whitespace-nowrap">
                  {formatAge(e.lastTimestamp ?? e.eventTime)} ago
                  {e.count > 1 && <span className="ml-2">×{e.count}</span>}
                </span>
              </div>
              {e.message && <p className="mt-1 text-xs text-zinc-400">{e.message}</p>}
              {e.source?.component && (
                <p className="mt-0.5 text-[10px] text-zinc-600">from {e.source.component}</p>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}

// ─── Children tab ─────────────────────────────────────────────────────────────

function ChildrenTab({ owned }: { owned: { kind: string; item: any }[] }) {
  if (owned.length === 0) {
    return (
      <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-8 text-center text-sm text-zinc-500">
        Nothing owns this resource via <code className="text-xs">ownerReferences</code>.
      </div>
    );
  }
  return (
    <div className="bg-zinc-900 border border-zinc-800 rounded-xl divide-y divide-zinc-800">
      {owned.map(({ kind, item }, i) => {
        const status = getStatus(item);
        return (
          <Link
            key={i}
            to={`${KIND_PATH[kind]}/${item.metadata?.namespace}/${item.metadata?.name}`}
            className="flex items-center justify-between px-4 py-3 hover:bg-zinc-800/30 transition-colors"
          >
            <div className="flex items-center gap-3 min-w-0">
              <span className="text-[10px] font-medium text-zinc-500 uppercase tracking-wider w-24 flex-shrink-0">
                {kind.replace(/^Talos/, '')}
              </span>
              <span className="text-sm text-zinc-200 font-medium truncate">{item.metadata?.name}</span>
            </div>
            <div className="flex items-center gap-2">
              <span className={`w-1.5 h-1.5 rounded-full ${TONE_DOT[status.tone]}`} />
              <span className={`text-xs ${TONE_TEXT[status.tone]}`}>{status.label}</span>
            </div>
          </Link>
        );
      })}
    </div>
  );
}
