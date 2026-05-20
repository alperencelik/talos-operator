import React, { useState, useEffect, useCallback, useRef } from 'react';
import axios from 'axios';
import {
  BrowserRouter,
  Routes,
  Route,
  useLocation,
  useNavigate,
  Navigate,
} from 'react-router-dom';
import {
  QueryClient,
  QueryClientProvider,
  useQuery,
} from '@tanstack/react-query';
import { RefreshCw } from 'lucide-react';
import Sidebar from './components/Sidebar';
import Overview from './components/Overview';
import ResourceList from './components/ResourceList';
import ResourceDetail from './components/ResourceDetail';
import TalosResourceForm from './components/TalosResourceForm';
import ClusterVisualizer from './components/ClusterVisualizer';
import './App.css';

export interface Resources {
  talosClusters: any[];
  talosControlPlanes: any[];
  talosWorkers: any[];
  talosMachines: any[];
  talosClusterAddons: any[];
  talosClusterAddonReleases: any[];
  talosEtcdBackups: any[];
  talosEtcdBackupSchedules: any[];
}

interface Toast {
  id: number;
  message: string;
  type: 'success' | 'error' | 'info';
}

const SECTION_TITLES: Record<string, string> = {
  '': 'Overview',
  clusters: 'Clusters',
  'control-planes': 'Control Planes',
  workers: 'Workers',
  machines: 'Machines',
  addons: 'Cluster Add-ons',
  'addon-releases': 'Add-on Releases',
  'etcd-backups': 'Etcd Backups',
  'etcd-backup-schedules': 'Etcd Backup Schedules',
  generator: 'Resource Generator',
  visualizer: 'Cluster Visualizer',
};

export const KIND_PATH: Record<string, string> = {
  TalosCluster: '/clusters',
  TalosControlPlane: '/control-planes',
  TalosWorker: '/workers',
  TalosMachine: '/machines',
  TalosClusterAddon: '/addons',
  TalosClusterAddonRelease: '/addon-releases',
  TalosEtcdBackup: '/etcd-backups',
  TalosEtcdBackupSchedule: '/etcd-backup-schedules',
};

function sectionTitle(pathname: string): string {
  const seg = pathname.split('/').filter(Boolean)[0] ?? '';
  return SECTION_TITLES[seg] ?? 'Talos Operator';
}

// Global query keys live here so every component agrees on what to invalidate.
export const queryKeys = {
  resources: () => ['resources'] as const,
  resource: (kind: string, namespace: string, name: string) =>
    ['resource', kind, namespace, name] as const,
  events: (kind: string, namespace: string, name: string) =>
    ['events', kind, namespace, name] as const,
};

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: true,
      staleTime: 2000,
    },
  },
});

function AppShell() {
  const [toasts, setToasts] = useState<Toast[]>([]);
  const toastIdRef = useRef(0);
  const failureToastedRef = useRef(false);
  const location = useLocation();
  const navigate = useNavigate();

  const addToast = useCallback((message: string, type: Toast['type'] = 'info') => {
    toastIdRef.current += 1;
    const id = toastIdRef.current;
    setToasts(prev => [...prev, { id, message, type }]);
    setTimeout(() => setToasts(prev => prev.filter(t => t.id !== id)), 5000);
  }, []);

  const resourcesQuery = useQuery<Resources>({
    queryKey: queryKeys.resources(),
    queryFn: async () => {
      const res = await axios.get('/api/resources');
      return res.data;
    },
    refetchInterval: 5000,
  });

  // One-shot connectivity toast on initial failure. Re-armed when we recover.
  useEffect(() => {
    if (resourcesQuery.isError && !failureToastedRef.current) {
      addToast('Failed to fetch resources from cluster', 'error');
      failureToastedRef.current = true;
    }
    if (resourcesQuery.isSuccess) {
      failureToastedRef.current = false;
    }
  }, [resourcesQuery.isError, resourcesQuery.isSuccess, addToast]);

  const resources = resourcesQuery.data ?? null;
  const loading = resourcesQuery.isLoading;
  const fetching = resourcesQuery.isFetching;
  const connected = resourcesQuery.isSuccess && !resourcesQuery.isError;

  // Manual refresh button: triggers a refetch of the resources query and
  // invalidates anything else (events, single-resource) that might be on
  // screen so the detail view's data refreshes too.
  const refreshAll = useCallback(() => {
    queryClient.invalidateQueries();
  }, []);

  const onActionSuccess = useCallback(
    (message: string) => addToast(message, 'success'),
    [addToast]
  );
  const onActionError = useCallback(
    (message: string) => addToast(message, 'error'),
    [addToast]
  );

  const listProps = {
    loading,
    onActionSuccess,
    onActionError,
    onRefresh: refreshAll,
  };

  const detailProps = {
    resources,
    loading,
    onActionSuccess,
    onActionError,
    onRefresh: refreshAll,
  };

  return (
    <div className="flex h-screen overflow-hidden bg-zinc-950 text-zinc-100 font-sans">
      <Sidebar connected={connected} />

      <div className="flex-1 min-w-0 flex flex-col overflow-hidden">
        <header className="flex-shrink-0 h-11 bg-zinc-900 border-b border-zinc-800 flex items-center justify-between px-6">
          <span className="text-sm font-medium text-zinc-200">{sectionTitle(location.pathname)}</span>
          <button
            onClick={refreshAll}
            disabled={fetching}
            className="flex items-center gap-1.5 text-xs text-zinc-500 hover:text-zinc-300 transition-colors px-2 py-1 rounded hover:bg-zinc-800 disabled:opacity-40 disabled:cursor-not-allowed"
          >
            <RefreshCw size={12} className={fetching ? 'animate-spin' : ''} />
            Refresh
          </button>
        </header>

        <main className="flex-1 overflow-auto">
          <Routes>
            <Route
              path="/"
              element={<Overview resources={resources} loading={loading} onNavigate={navigate} />}
            />

            <Route
              path="/clusters"
              element={<ResourceList title="Clusters" kind="TalosCluster" items={resources?.talosClusters ?? []} {...listProps} />}
            />
            <Route path="/clusters/:namespace/:name" element={<ResourceDetail kind="TalosCluster" {...detailProps} />} />

            <Route
              path="/control-planes"
              element={<ResourceList title="Control Planes" kind="TalosControlPlane" items={resources?.talosControlPlanes ?? []} {...listProps} />}
            />
            <Route path="/control-planes/:namespace/:name" element={<ResourceDetail kind="TalosControlPlane" {...detailProps} />} />

            <Route
              path="/workers"
              element={<ResourceList title="Workers" kind="TalosWorker" items={resources?.talosWorkers ?? []} {...listProps} />}
            />
            <Route path="/workers/:namespace/:name" element={<ResourceDetail kind="TalosWorker" {...detailProps} />} />

            <Route
              path="/machines"
              element={<ResourceList title="Machines" kind="TalosMachine" items={resources?.talosMachines ?? []} {...listProps} />}
            />
            <Route path="/machines/:namespace/:name" element={<ResourceDetail kind="TalosMachine" {...detailProps} />} />

            <Route
              path="/addons"
              element={<ResourceList title="Cluster Add-ons" kind="TalosClusterAddon" items={resources?.talosClusterAddons ?? []} {...listProps} />}
            />
            <Route path="/addons/:namespace/:name" element={<ResourceDetail kind="TalosClusterAddon" {...detailProps} />} />

            <Route
              path="/addon-releases"
              element={<ResourceList title="Add-on Releases" kind="TalosClusterAddonRelease" items={resources?.talosClusterAddonReleases ?? []} {...listProps} />}
            />
            <Route path="/addon-releases/:namespace/:name" element={<ResourceDetail kind="TalosClusterAddonRelease" {...detailProps} />} />

            <Route
              path="/etcd-backups"
              element={<ResourceList title="Etcd Backups" kind="TalosEtcdBackup" items={resources?.talosEtcdBackups ?? []} {...listProps} />}
            />
            <Route path="/etcd-backups/:namespace/:name" element={<ResourceDetail kind="TalosEtcdBackup" {...detailProps} />} />

            <Route
              path="/etcd-backup-schedules"
              element={<ResourceList title="Etcd Backup Schedules" kind="TalosEtcdBackupSchedule" items={resources?.talosEtcdBackupSchedules ?? []} {...listProps} />}
            />
            <Route path="/etcd-backup-schedules/:namespace/:name" element={<ResourceDetail kind="TalosEtcdBackupSchedule" {...detailProps} />} />

            <Route
              path="/generator"
              element={
                <TalosResourceForm
                  onApplySuccess={() => {
                    addToast('Resource applied successfully', 'success');
                    refreshAll();
                  }}
                  onApplyError={onActionError}
                />
              }
            />
            <Route
              path="/visualizer"
              element={<ClusterVisualizer resources={resources} loading={loading} onRefresh={refreshAll} />}
            />

            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </main>
      </div>

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

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <AppShell />
      </BrowserRouter>
    </QueryClientProvider>
  );
}
