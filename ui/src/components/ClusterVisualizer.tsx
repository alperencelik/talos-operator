import React, { useState, useEffect, useCallback } from 'react';
import ReactFlow, {
  Node,
  Edge,
  useNodesState,
  useEdgesState,
  Controls,
  Background,
  BackgroundVariant,
} from 'reactflow';
import 'reactflow/dist/style.css';
import YAML from 'js-yaml';
import dagre from 'dagre';
import { X, RefreshCw, FileCode } from 'lucide-react';
import { Resources } from '../App';
import { useEscape } from '../hooks/useEscape';

// ─── Status derivation (mirrors ResourceList) ─────────────────────────────────

type StatusTone = 'green' | 'red' | 'yellow' | 'gray';

const IN_PROGRESS_REASONS = new Set([
  'pending', 'creating', 'initializing', 'provisioning',
  'reconciling', 'inprogress', 'progressing', 'updating', 'upgrading',
]);

const TONE_DOT_COLOR: Record<StatusTone, string> = {
  green: '#22c55e',
  red: '#ef4444',
  yellow: '#eab308',
  gray: '#52525b',
};

function deriveTone(item: any): { tone: StatusTone; tooltip?: string } {
  const conditions: any[] = item?.status?.conditions ?? [];
  if (conditions.length === 0) return { tone: 'gray' };
  const ready = conditions.find((c: any) => c.type === 'Ready');
  if (ready) {
    const tooltip = ready.message || ready.reason;
    if (ready.status === 'True') return { tone: 'green', tooltip };
    if (ready.status === 'Unknown') return { tone: 'yellow', tooltip };
    const reasonLower = (ready.reason ?? '').toLowerCase();
    return { tone: IN_PROGRESS_REASONS.has(reasonLower) ? 'yellow' : 'red', tooltip };
  }
  const failing = conditions.find((c: any) => c.status === 'False');
  if (failing) {
    return { tone: 'red', tooltip: failing.message || `${failing.type}: ${failing.reason ?? 'False'}` };
  }
  return { tone: 'gray' };
}

// ─── Dagre layout ─────────────────────────────────────────────────────────────

function getLayoutedElements(nodes: Node[], edges: Edge[]) {
  const g = new dagre.graphlib.Graph();
  g.setGraph({ rankdir: 'TB', ranksep: 80, nodesep: 60 });
  g.setDefaultNodeLabel(() => ({}));
  g.setDefaultEdgeLabel(() => ({}));

  nodes.forEach(node => g.setNode(node.id, { width: 220, height: 80 }));
  edges.forEach(edge => g.setEdge(edge.source, edge.target));
  dagre.layout(g);

  const layouted = nodes.map(node => {
    const pos = g.node(node.id);
    return { ...node, position: { x: pos.x - 110, y: pos.y - 40 } };
  });

  // Align TalosWorker nodes with their referenced TalosControlPlane row,
  // and push owned TalosMachine nodes immediately below their owner.
  layouted.forEach(node => {
    if (node.data.kind === 'TalosWorker' && node.data.spec?.controlPlaneRef?.name) {
      const cpNode = layouted.find(
        n => n.data.kind === 'TalosControlPlane' && n.id === node.data.spec.controlPlaneRef.name
      );
      if (cpNode) {
        node.position.y = cpNode.position.y;
        layouted.forEach(machine => {
          if (
            machine.data.kind === 'TalosMachine' &&
            machine.data.metadata?.ownerReferences?.some(
              (ref: any) => ref.kind === 'TalosWorker' && ref.name === node.id
            )
          ) {
            machine.position.y = node.position.y + 130;
          }
        });
      }
    }
  });

  return layouted;
}

// ─── Node styles ──────────────────────────────────────────────────────────────

interface KindStyle {
  background: string;
  border: string;
}

const KIND_STYLES: Record<string, KindStyle> = {
  TalosCluster: { background: 'rgba(255,107,53,0.08)', border: 'rgba(255,107,53,0.5)' },
  TalosControlPlane: { background: 'rgba(56,189,248,0.06)', border: 'rgba(56,189,248,0.35)' },
  TalosWorker: { background: 'rgba(192,132,252,0.06)', border: 'rgba(192,132,252,0.35)' },
  TalosMachine: { background: '#18181b', border: '#3f3f46' },
  TalosClusterAddon: { background: 'rgba(129,140,248,0.07)', border: 'rgba(129,140,248,0.4)' },
  TalosClusterAddonRelease: { background: 'rgba(129,140,248,0.05)', border: 'rgba(129,140,248,0.3)' },
  TalosEtcdBackup: { background: 'rgba(245,158,11,0.07)', border: 'rgba(245,158,11,0.4)' },
  TalosEtcdBackupSchedule: { background: 'rgba(245,158,11,0.05)', border: 'rgba(245,158,11,0.3)' },
};

function nodeStyle(kind: string, tone: StatusTone): React.CSSProperties {
  const base = KIND_STYLES[kind] ?? KIND_STYLES.TalosMachine;
  // Overlay a status-tinted left border for at-a-glance health.
  return {
    background: base.background,
    border: `1px solid ${base.border}`,
    borderLeft: `3px solid ${TONE_DOT_COLOR[tone]}`,
    borderRadius: '10px',
    color: '#e4e4e7',
    fontSize: '12px',
    padding: '8px 12px',
    width: 220,
    fontFamily: 'Inter, sans-serif',
  };
}

function nodeLabel(kind: string, name: string, tone: StatusTone) {
  return (
    <div className="flex items-center gap-1.5 text-left">
      <span
        className="inline-block w-1.5 h-1.5 rounded-full flex-shrink-0"
        style={{ background: TONE_DOT_COLOR[tone] }}
      />
      <div className="min-w-0">
        <div className="text-[10px] uppercase tracking-wider text-zinc-500 leading-none">
          {kind.replace(/^Talos/, '')}
        </div>
        <div className="text-[12px] text-zinc-100 truncate">{name}</div>
      </div>
    </div>
  );
}

// ─── YAML modal ───────────────────────────────────────────────────────────────

function YamlModal({ node, onClose }: { node: any; onClose: () => void }) {
  useEscape(onClose);
  const yaml = YAML.dump(node);
  return (
    <div
      className="absolute inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center z-50 p-4"
      onClick={onClose}
      role="dialog"
      aria-modal="true"
    >
      <div
        className="bg-zinc-900 border border-zinc-700 rounded-xl shadow-2xl w-full max-w-xl max-h-[75vh] flex flex-col"
        onClick={e => e.stopPropagation()}
      >
        <div className="flex items-center justify-between px-5 py-3.5 border-b border-zinc-800 flex-shrink-0">
          <div className="flex items-center gap-2 min-w-0">
            <FileCode size={14} className="text-zinc-500 flex-shrink-0" />
            <span className="text-sm font-semibold text-zinc-100 truncate">
              {node?.metadata?.name}
            </span>
          </div>
          <button
            onClick={onClose}
            className="text-zinc-500 hover:text-zinc-300 transition-colors ml-3 p-1 hover:bg-zinc-800 rounded"
            aria-label="Close"
          >
            <X size={14} />
          </button>
        </div>
        <div className="overflow-auto flex-1 p-5">
          <pre className="text-xs font-mono text-zinc-300 leading-relaxed whitespace-pre">{yaml}</pre>
        </div>
      </div>
    </div>
  );
}

// ─── Main component ───────────────────────────────────────────────────────────

interface ClusterVisualizerProps {
  resources: Resources | null;
  loading: boolean;
  onRefresh: () => void;
}

const edgeBase = {
  animated: true,
  style: { stroke: '#52525b', strokeWidth: 1.5 },
  labelStyle: { fontSize: 10, fill: '#71717a' },
  labelBgStyle: { fill: '#18181b', fillOpacity: 0.8 },
};

export default function ClusterVisualizer({ resources, loading, onRefresh }: ClusterVisualizerProps) {
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [selectedNode, setSelectedNode] = useState<any>(null);

  const buildGraph = useCallback(() => {
    if (!resources) return;

    const {
      talosClusters,
      talosControlPlanes,
      talosWorkers,
      talosMachines,
      talosClusterAddons,
      talosClusterAddonReleases,
      talosEtcdBackups,
      talosEtcdBackupSchedules,
    } = resources;

    const newNodes: Node[] = [];
    const newEdges: Edge[] = [];

    function pushNode(item: any, kind: string) {
      const { tone, tooltip } = deriveTone(item);
      newNodes.push({
        id: item.metadata?.name,
        type: 'default',
        data: { label: nodeLabel(kind, item.metadata?.name ?? '', tone), kind, tone, tooltip, ...item },
        position: { x: 0, y: 0 },
        style: nodeStyle(kind, tone),
      });
    }

    talosClusters.forEach(c => pushNode(c, 'TalosCluster'));
    talosControlPlanes.forEach(cp => pushNode(cp, 'TalosControlPlane'));
    talosWorkers.forEach(w => pushNode(w, 'TalosWorker'));
    talosMachines.forEach(m => pushNode(m, 'TalosMachine'));
    talosClusterAddons.forEach(a => pushNode(a, 'TalosClusterAddon'));
    talosClusterAddonReleases.forEach(r => pushNode(r, 'TalosClusterAddonRelease'));
    talosEtcdBackups.forEach(b => pushNode(b, 'TalosEtcdBackup'));
    talosEtcdBackupSchedules.forEach(s => pushNode(s, 'TalosEtcdBackupSchedule'));

    // ── Edges: cluster topology ─────────────────────────────────────────────
    talosClusters.forEach(c => {
      if (c.spec?.controlPlaneRef)
        newEdges.push({ ...edgeBase, id: `e-${c.metadata.name}-cp-ref`, source: c.metadata.name, target: c.spec.controlPlaneRef.name, label: 'ref' });
      if (c.spec?.workerRef)
        newEdges.push({ ...edgeBase, id: `e-${c.metadata.name}-wk-ref`, source: c.metadata.name, target: c.spec.workerRef.name, label: 'ref' });
    });

    talosControlPlanes.forEach(cp => {
      const owner = cp.metadata?.ownerReferences?.find((r: any) => r.kind === 'TalosCluster');
      if (owner)
        newEdges.push({ ...edgeBase, id: `e-${owner.name}-${cp.metadata.name}-own`, source: owner.name, target: cp.metadata.name, label: 'owns' });
    });

    talosWorkers.forEach(w => {
      if (w.spec?.controlPlaneRef)
        newEdges.push({ ...edgeBase, id: `e-${w.metadata.name}-cp`, source: w.metadata.name, target: w.spec.controlPlaneRef.name, label: 'ref', type: 'smoothstep' });
      const owner = w.metadata?.ownerReferences?.find((r: any) => r.kind === 'TalosCluster');
      if (owner)
        newEdges.push({ ...edgeBase, id: `e-${owner.name}-${w.metadata.name}-own`, source: owner.name, target: w.metadata.name, label: 'owns' });
    });

    talosMachines.forEach(m => {
      if (m.metadata?.ownerReferences?.length) {
        const owner = m.metadata.ownerReferences[0].name;
        newEdges.push({ ...edgeBase, id: `e-${owner}-${m.metadata.name}`, source: owner, target: m.metadata.name, label: 'owns' });
      }
    });

    // ── Edges: add-ons / releases ───────────────────────────────────────────
    talosClusterAddonReleases.forEach(r => {
      const clusterName = r.spec?.clusterRef?.name;
      if (clusterName)
        newEdges.push({ ...edgeBase, id: `e-${r.metadata.name}-cluster`, source: clusterName, target: r.metadata.name, label: 'installs' });
      const owner = r.metadata?.ownerReferences?.find((o: any) => o.kind === 'TalosClusterAddon');
      if (owner)
        newEdges.push({ ...edgeBase, id: `e-${owner.name}-${r.metadata.name}-own`, source: owner.name, target: r.metadata.name, label: 'owns' });
    });

    // ── Edges: etcd backups ─────────────────────────────────────────────────
    talosEtcdBackups.forEach(b => {
      const cp = b.spec?.talosControlPlaneRef?.name;
      if (cp)
        newEdges.push({ ...edgeBase, id: `e-${b.metadata.name}-cp`, source: cp, target: b.metadata.name, label: 'backs up' });
      const owner = b.metadata?.ownerReferences?.find((o: any) => o.kind === 'TalosEtcdBackupSchedule');
      if (owner)
        newEdges.push({ ...edgeBase, id: `e-${owner.name}-${b.metadata.name}-own`, source: owner.name, target: b.metadata.name, label: 'owns' });
    });

    talosEtcdBackupSchedules.forEach(s => {
      const cp = s.spec?.backupTemplate?.spec?.talosControlPlaneRef?.name;
      if (cp)
        newEdges.push({ ...edgeBase, id: `e-${s.metadata.name}-cp`, source: cp, target: s.metadata.name, label: 'schedules' });
    });

    setNodes(getLayoutedElements(newNodes, newEdges));
    setEdges(newEdges);
  }, [resources, setNodes, setEdges]);

  useEffect(() => {
    buildGraph();
  }, [buildGraph]);

  const onNodeClick = useCallback((_: React.MouseEvent, node: Node) => {
    setSelectedNode(node.data);
  }, []);

  const onNodeDrag = useCallback(
    (_: any, draggedNode: Node) => {
      setNodes(nds =>
        nds.map(node => {
          if (node.id !== draggedNode.id) return node;

          let { x, y } = draggedNode.position;
          const dw = node.width ?? 220;
          const dh = node.height ?? 80;

          nds.forEach(other => {
            if (other.id === draggedNode.id) return;
            const ow = other.width ?? 220;
            const oh = other.height ?? 80;
            const op = other.position;
            const overlapX = Math.min(x + dw, op.x + ow) - Math.max(x, op.x);
            const overlapY = Math.min(y + dh, op.y + oh) - Math.max(y, op.y);
            if (overlapX > 0 && overlapY > 0) {
              if (overlapX < overlapY) x = x < op.x ? op.x - dw : op.x + ow;
              else y = y < op.y ? op.y - dh : op.y + oh;
            }
          });

          return { ...node, position: { x, y } };
        })
      );
    },
    [setNodes]
  );

  const isEmpty = !resources ||
    (resources.talosClusters.length +
      resources.talosControlPlanes.length +
      resources.talosWorkers.length +
      resources.talosMachines.length +
      resources.talosClusterAddons.length +
      resources.talosClusterAddonReleases.length +
      resources.talosEtcdBackups.length +
      resources.talosEtcdBackupSchedules.length === 0);

  return (
    <div className="flex flex-col h-full relative">
      {/* Toolbar */}
      <div className="flex-shrink-0 px-5 py-3 border-b border-zinc-800 flex items-center justify-between bg-zinc-950">
        <div className="flex items-center gap-4">
          <p className="text-xs text-zinc-500">
            Click a node to inspect its YAML. The colored bar shows Ready state.
          </p>
          <div className="flex items-center gap-3 text-[10px] text-zinc-600">
            {(['green', 'yellow', 'red', 'gray'] as StatusTone[]).map(t => (
              <span key={t} className="flex items-center gap-1">
                <span className="w-1.5 h-1.5 rounded-full" style={{ background: TONE_DOT_COLOR[t] }} />
                {t === 'green' ? 'Ready' : t === 'yellow' ? 'Pending' : t === 'red' ? 'Failing' : 'Unknown'}
              </span>
            ))}
          </div>
        </div>
        <button
          onClick={onRefresh}
          disabled={loading}
          className="flex items-center gap-1.5 text-xs text-zinc-500 hover:text-zinc-300 transition-colors px-2 py-1 rounded hover:bg-zinc-800 disabled:opacity-40"
        >
          <RefreshCw size={12} className={loading ? 'animate-spin' : ''} />
          Reload graph
        </button>
      </div>

      {/* ReactFlow canvas */}
      <div className="flex-1 relative">
        {loading ? (
          <div className="flex items-center justify-center h-full gap-3 text-sm text-zinc-500">
            <div className="w-4 h-4 border-2 border-zinc-700 border-t-brand rounded-full animate-spin" />
            Loading cluster topology…
          </div>
        ) : isEmpty ? (
          <div className="flex flex-col items-center justify-center h-full text-center px-8">
            <div className="text-zinc-600 text-sm mb-1">No resources to visualize</div>
            <div className="text-zinc-700 text-xs max-w-xs">
              Create Talos resources using the Generator and they'll appear here as a graph.
            </div>
          </div>
        ) : (
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onNodeClick={onNodeClick}
            onNodeDrag={onNodeDrag}
            fitView
            fitViewOptions={{ padding: 0.3 }}
          >
            <Controls />
            <Background variant={BackgroundVariant.Dots} gap={20} size={1} color="#27272a" />
          </ReactFlow>
        )}

        {/* YAML modal */}
        {selectedNode && (
          <YamlModal node={selectedNode} onClose={() => setSelectedNode(null)} />
        )}
      </div>
    </div>
  );
}
