import React, { useState, useEffect, useCallback } from 'react';
import axios from 'axios';
import { RefreshCw } from 'lucide-react';
import Sidebar from './components/Sidebar';
import Overview from './components/Overview';
import ResourceList from './components/ResourceList';
import TalosResourceForm from './components/TalosResourceForm';
import ClusterVisualizer from './components/ClusterVisualizer';
import './App.css';

export type Page =
  | 'overview'
  | 'clusters'
  | 'control-planes'
  | 'workers'
  | 'machines'
  | 'generator'
  | 'visualizer';

export interface Resources {
  talosClusters: any[];
  talosControlPlanes: any[];
  talosWorkers: any[];
  talosMachines: any[];
}

interface Toast {
  id: number;
  message: string;
  type: 'success' | 'error' | 'info';
}

const PAGE_TITLES: Record<Page, string> = {
  overview: 'Overview',
  clusters: 'Clusters',
  'control-planes': 'Control Planes',
  workers: 'Workers',
  machines: 'Machines',
  generator: 'Resource Generator',
  visualizer: 'Cluster Visualizer',
};

function App() {
  const [page, setPage] = useState<Page>('overview');
  const [resources, setResources] = useState<Resources | null>(null);
  const [loading, setLoading] = useState(false);
  const [toasts, setToasts] = useState<Toast[]>([]);
  const [connected, setConnected] = useState(false);

  const addToast = useCallback((message: string, type: Toast['type'] = 'info') => {
    const id = Date.now();
    setToasts(prev => [...prev, { id, message, type }]);
    setTimeout(() => setToasts(prev => prev.filter(t => t.id !== id)), 5000);
  }, []);

  const fetchResources = useCallback(async () => {
    setLoading(true);
    try {
      const res = await axios.get('/api/resources');
      setResources(res.data);
      setConnected(true);
    } catch {
      setConnected(false);
      addToast('Failed to fetch resources from cluster', 'error');
    } finally {
      setLoading(false);
    }
  }, [addToast]);

  useEffect(() => {
    fetchResources();
  }, [fetchResources]);

  const navigate = useCallback(
    (newPage: Page) => {
      setPage(newPage);
      if (newPage !== 'generator') {
        fetchResources();
      }
    },
    [fetchResources]
  );

  const renderPage = () => {
    switch (page) {
      case 'overview':
        return (
          <Overview resources={resources} loading={loading} onNavigate={navigate} />
        );
      case 'clusters':
        return (
          <ResourceList
            title="Clusters"
            kind="TalosCluster"
            items={resources?.talosClusters ?? []}
            loading={loading}
          />
        );
      case 'control-planes':
        return (
          <ResourceList
            title="Control Planes"
            kind="TalosControlPlane"
            items={resources?.talosControlPlanes ?? []}
            loading={loading}
          />
        );
      case 'workers':
        return (
          <ResourceList
            title="Workers"
            kind="TalosWorker"
            items={resources?.talosWorkers ?? []}
            loading={loading}
          />
        );
      case 'machines':
        return (
          <ResourceList
            title="Machines"
            kind="TalosMachine"
            items={resources?.talosMachines ?? []}
            loading={loading}
          />
        );
      case 'generator':
        return (
          <TalosResourceForm
            onApplySuccess={() => {
              addToast('Resource applied successfully', 'success');
              fetchResources();
            }}
            onApplyError={(msg: string) => addToast(msg, 'error')}
          />
        );
      case 'visualizer':
        return <ClusterVisualizer resources={resources} loading={loading} onRefresh={fetchResources} />;
      default:
        return null;
    }
  };

  return (
    <div className="flex h-screen overflow-hidden bg-zinc-950 text-zinc-100 font-sans">
      {/* Sidebar */}
      <Sidebar currentPage={page} onNavigate={navigate} connected={connected} />

      {/* Main content */}
      <div className="flex-1 min-w-0 flex flex-col overflow-hidden">
        {/* Top bar */}
        <header className="flex-shrink-0 h-11 bg-zinc-900 border-b border-zinc-800 flex items-center justify-between px-6">
          <span className="text-sm font-medium text-zinc-200">{PAGE_TITLES[page]}</span>
          <button
            onClick={fetchResources}
            disabled={loading}
            className="flex items-center gap-1.5 text-xs text-zinc-500 hover:text-zinc-300 transition-colors px-2 py-1 rounded hover:bg-zinc-800 disabled:opacity-40 disabled:cursor-not-allowed"
          >
            <RefreshCw size={12} className={loading ? 'animate-spin' : ''} />
            Refresh
          </button>
        </header>

        {/* Page content */}
        <main className="flex-1 overflow-auto">{renderPage()}</main>
      </div>

      {/* Toast notifications */}
      <div className="fixed bottom-4 right-4 flex flex-col gap-2 z-50 pointer-events-none">
        {toasts.map(toast => (
          <div
            key={toast.id}
            className={`px-4 py-3 rounded-lg text-sm font-medium shadow-xl border pointer-events-auto max-w-sm ${
              toast.type === 'success'
                ? 'bg-green-950 text-green-200 border-green-900'
                : toast.type === 'error'
                ? 'bg-red-950 text-red-200 border-red-900'
                : 'bg-zinc-800 text-zinc-200 border-zinc-700'
            }`}
          >
            {toast.message}
          </div>
        ))}
      </div>
    </div>
  );
}

export default App;
