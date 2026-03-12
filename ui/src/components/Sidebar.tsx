import React from 'react';
import {
  LayoutDashboard,
  Server,
  Cpu,
  Layers,
  HardDrive,
  Wand2,
  GitBranch,
  Circle,
} from 'lucide-react';
import { Page } from '../App';

interface NavItem {
  id: Page;
  label: string;
  icon: React.ReactNode;
}

interface NavGroup {
  label?: string;
  items: NavItem[];
}

const navGroups: NavGroup[] = [
  {
    items: [
      { id: 'overview', label: 'Overview', icon: <LayoutDashboard size={15} /> },
    ],
  },
  {
    label: 'Resources',
    items: [
      { id: 'clusters', label: 'Clusters', icon: <Server size={15} /> },
      { id: 'control-planes', label: 'Control Planes', icon: <Cpu size={15} /> },
      { id: 'workers', label: 'Workers', icon: <Layers size={15} /> },
      { id: 'machines', label: 'Machines', icon: <HardDrive size={15} /> },
    ],
  },
  {
    label: 'Tools',
    items: [
      { id: 'generator', label: 'Generator', icon: <Wand2 size={15} /> },
      { id: 'visualizer', label: 'Visualizer', icon: <GitBranch size={15} /> },
    ],
  },
];

interface SidebarProps {
  currentPage: Page;
  onNavigate: (page: Page) => void;
  connected: boolean;
}

export default function Sidebar({ currentPage, onNavigate, connected }: SidebarProps) {
  return (
    <div className="w-56 flex-shrink-0 bg-zinc-900 border-r border-zinc-800 flex flex-col select-none">
      {/* Logo */}
      <div className="px-4 py-4 border-b border-zinc-800">
        <div className="flex items-center gap-2.5">
          <img src="/logo.png" alt="Talos" className="w-7 h-7 rounded" />
          <div>
            <div className="text-zinc-100 text-sm font-semibold leading-tight">Talos Operator</div>
            <div className="text-zinc-500 text-xs">Dashboard</div>
          </div>
        </div>
      </div>

      {/* Navigation */}
      <nav className="flex-1 px-2 py-3 overflow-y-auto space-y-5">
        {navGroups.map((group, gi) => (
          <div key={gi}>
            {group.label && (
              <div className="px-2 mb-1 text-[10px] font-semibold uppercase tracking-wider text-zinc-600">
                {group.label}
              </div>
            )}
            <div className="space-y-0.5">
              {group.items.map(item => {
                const active = currentPage === item.id;
                return (
                  <button
                    key={item.id}
                    onClick={() => onNavigate(item.id)}
                    className={`w-full flex items-center gap-2.5 px-2.5 py-2 rounded-md text-sm transition-colors text-left ${
                      active
                        ? 'bg-brand-dim text-brand font-medium'
                        : 'text-zinc-400 hover:text-zinc-100 hover:bg-zinc-800'
                    }`}
                  >
                    <span className={active ? 'text-brand' : 'text-zinc-500'}>
                      {item.icon}
                    </span>
                    {item.label}
                  </button>
                );
              })}
            </div>
          </div>
        ))}
      </nav>

      {/* Connection status */}
      <div className="px-4 py-3 border-t border-zinc-800">
        <div className="flex items-center gap-2">
          <Circle
            size={7}
            className={connected ? 'fill-green-500 text-green-500' : 'fill-red-500 text-red-500'}
          />
          <span className="text-xs text-zinc-500">
            {connected ? 'Connected' : 'Disconnected'}
          </span>
        </div>
      </div>
    </div>
  );
}
