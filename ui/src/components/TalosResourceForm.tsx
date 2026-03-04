import React, { useState, useEffect } from 'react';
import YAML from 'js-yaml';
import axios from 'axios';
import { Copy, Download, Send, Check, ChevronDown } from 'lucide-react';
import 'reactflow/dist/style.css';
import '../TalosUI.css';

// ─── Styled primitives ────────────────────────────────────────────────────────

const inputCls =
  'w-full bg-zinc-950 border border-zinc-700 rounded-md px-3 py-2 text-sm text-zinc-100 ' +
  'placeholder:text-zinc-600 focus:outline-none focus:border-brand focus:ring-1 focus:ring-brand ' +
  'transition-colors';

const selectCls =
  'w-full bg-zinc-950 border border-zinc-700 rounded-md px-3 py-2 text-sm text-zinc-100 ' +
  'focus:outline-none focus:border-brand focus:ring-1 focus:ring-brand transition-colors appearance-none cursor-pointer';

const textareaCls =
  'w-full bg-zinc-950 border border-zinc-700 rounded-md px-3 py-2 text-sm text-zinc-100 font-mono ' +
  'placeholder:text-zinc-600 focus:outline-none focus:border-brand focus:ring-1 focus:ring-brand ' +
  'transition-colors resize-none';

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="mb-4">
      <label className="block text-xs font-medium text-zinc-400 mb-1.5">{label}</label>
      {children}
    </div>
  );
}

function SelectField({
  label,
  value,
  onChange,
  options,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  options: { value: string; label: string }[];
}) {
  return (
    <Field label={label}>
      <div className="relative">
        <select className={selectCls} value={value} onChange={e => onChange(e.target.value)}>
          {options.map(o => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
        <ChevronDown
          size={13}
          className="absolute right-2.5 top-1/2 -translate-y-1/2 text-zinc-500 pointer-events-none"
        />
      </div>
    </Field>
  );
}

function Divider({ label }: { label: string }) {
  return (
    <div className="flex items-center gap-3 my-5">
      <div className="flex-1 h-px bg-zinc-800" />
      <span className="text-xs font-medium text-zinc-500 uppercase tracking-wider">{label}</span>
      <div className="flex-1 h-px bg-zinc-800" />
    </div>
  );
}

// ─── Main component ───────────────────────────────────────────────────────────

interface TalosResourceFormProps {
  onApplySuccess: () => void;
  onApplyError: (msg: string) => void;
}

function TalosResourceForm({ onApplySuccess, onApplyError }: TalosResourceFormProps) {
  // ── State ──────────────────────────────────────────────────────────────────
  const [resourceType, setResourceType] = useState('TalosCluster');

  // Common
  const [name, setName] = useState('sample');
  const [namespace, setNamespace] = useState('default');
  const [mode, setMode] = useState('metal');
  const [talosVersion, setTalosVersion] = useState('v1.10.3');
  const [kubernetesVersion, setKubernetesVersion] = useState('v1.31.0');
  const [replicas, setReplicas] = useState(2);
  const [machines, setMachines] = useState('<machine-ip-1>\n<machine-ip-2>');

  // TalosCluster
  const [talosClusterDefinitionMode, setTalosClusterDefinitionMode] = useState('inline');
  const [controlPlaneRef, setControlPlaneRef] = useState('taloscontrolplane-sample');
  const [workerRef, setWorkerRef] = useState('talosworker-sample');

  // Inline Control Plane
  const [inlineCPTalosVersion, setInlineCPTalosVersion] = useState('v1.10.3');
  const [inlineCPKubernetesVersion, setInlineCPKubernetesVersion] = useState('v1.31.0');
  const [inlineCPEndpoint, setInlineCPEndpoint] = useState('https://<control-plane-endpoint>:6443');
  const [inlineCPMachines, setInlineCPMachines] = useState(
    '<control-plane-machine-ip-1>\n<control-plane-machine-ip-2>'
  );
  const [inlineCPReplicas, setInlineCPReplicas] = useState(2);

  // Inline Worker
  const [inlineWKTalosVersion, setInlineWKTalosVersion] = useState('v1.10.3');
  const [inlineWKKubernetesVersion, setInlineWKKubernetesVersion] = useState('v1.31.0');
  const [inlineWKMachines, setInlineWKMachines] = useState(
    '<worker-machine-ip-1>\n<worker-machine-ip-2>'
  );
  const [inlineWKReplicas, setInlineWKReplicas] = useState(2);

  // TalosControlPlane
  const [controlPlaneEndpoint, setControlPlaneEndpoint] = useState(
    'https://<control-plane-endpoint>:6443'
  );

  // TalosWorker
  const [workerControlPlaneRef, setWorkerControlPlaneRef] = useState('taloscontrolplane-sample');

  // UI state
  const [generatedYaml, setGeneratedYaml] = useState('');
  const [copyLabel, setCopyLabel] = useState<'copy' | 'copied'>('copy');
  const [applyState, setApplyState] = useState<'idle' | 'loading' | 'success'>('idle');

  // ── Reset on resource type change ──────────────────────────────────────────
  useEffect(() => {
    setName('sample');
    setNamespace('default');
    setMode('metal');
    setTalosVersion('v1.10.3');
    setKubernetesVersion('v1.31.0');
    setReplicas(2);
    setMachines('<machine-ip-1>\n<machine-ip-2>');
    setControlPlaneRef('taloscontrolplane-sample');
    setWorkerRef('talosworker-sample');
    setControlPlaneEndpoint('https://<control-plane-endpoint>:6443');
    setWorkerControlPlaneRef('taloscontrolplane-sample');
    setTalosClusterDefinitionMode('inline');
    setInlineCPTalosVersion('v1.10.3');
    setInlineCPKubernetesVersion('v1.31.0');
    setInlineCPEndpoint('https://<control-plane-endpoint>:6443');
    setInlineCPMachines('<control-plane-machine-ip-1>\n<control-plane-machine-ip-2>');
    setInlineCPReplicas(2);
    setInlineWKTalosVersion('v1.10.3');
    setInlineWKKubernetesVersion('v1.31.0');
    setInlineWKMachines('<worker-machine-ip-1>\n<worker-machine-ip-2>');
    setInlineWKReplicas(2);
  }, [resourceType]);

  // ── YAML generation ────────────────────────────────────────────────────────
  useEffect(() => {
    const apiVersion = 'talos.alperen.cloud/v1alpha1';
    let resource: any;

    switch (resourceType) {
      case 'TalosCluster':
        if (talosClusterDefinitionMode === 'reference') {
          resource = {
            apiVersion,
            kind: 'TalosCluster',
            metadata: { name, namespace },
            spec: {
              controlPlaneRef: { name: controlPlaneRef },
              workerRef: { name: workerRef },
            },
          };
        } else {
          const inlineCPSpec: any = {
            version: inlineCPTalosVersion,
            kubeVersion: inlineCPKubernetesVersion,
            mode,
          };
          if (mode === 'metal') {
            inlineCPSpec.endpoint = inlineCPEndpoint;
            inlineCPSpec.metalSpec = {
              machines: inlineCPMachines.split('\n').filter(m => m.trim()),
            };
          } else {
            inlineCPSpec.replicas = inlineCPReplicas;
          }

          const inlineWKSpec: any = {
            version: inlineWKTalosVersion,
            kubeVersion: inlineWKKubernetesVersion,
            mode,
            controlPlaneRef: { name: `${name}-controlplane` },
          };
          if (mode === 'metal') {
            inlineWKSpec.metalSpec = {
              machines: inlineWKMachines.split('\n').filter(m => m.trim()),
            };
          } else {
            inlineWKSpec.replicas = inlineWKReplicas;
          }

          resource = {
            apiVersion,
            kind: 'TalosCluster',
            metadata: { name, namespace },
            spec: { controlPlane: inlineCPSpec, worker: inlineWKSpec },
          };
        }
        break;

      case 'TalosControlPlane': {
        const cpSpec: any = { version: talosVersion, kubeVersion: kubernetesVersion, mode };
        if (mode === 'metal') {
          cpSpec.endpoint = controlPlaneEndpoint;
          cpSpec.metalSpec = { machines: machines.split('\n').filter(m => m.trim()) };
        } else {
          cpSpec.replicas = replicas;
        }
        resource = {
          apiVersion,
          kind: 'TalosControlPlane',
          metadata: { name, namespace },
          spec: cpSpec,
        };
        break;
      }

      case 'TalosWorker': {
        const workerSpec: any = {
          version: talosVersion,
          kubeVersion: kubernetesVersion,
          mode,
          controlPlaneRef: { name: workerControlPlaneRef },
        };
        if (mode === 'metal') {
          workerSpec.metalSpec = { machines: machines.split('\n').filter(m => m.trim()) };
        } else {
          workerSpec.replicas = replicas;
        }
        resource = {
          apiVersion,
          kind: 'TalosWorker',
          metadata: { name, namespace },
          spec: workerSpec,
        };
        break;
      }

      default:
        resource = {};
    }

    setGeneratedYaml(YAML.dump(resource));
  }, [
    resourceType,
    name,
    namespace,
    mode,
    talosVersion,
    kubernetesVersion,
    machines,
    replicas,
    controlPlaneEndpoint,
    controlPlaneRef,
    workerRef,
    workerControlPlaneRef,
    talosClusterDefinitionMode,
    inlineCPTalosVersion,
    inlineCPKubernetesVersion,
    inlineCPEndpoint,
    inlineCPMachines,
    inlineCPReplicas,
    inlineWKTalosVersion,
    inlineWKKubernetesVersion,
    inlineWKMachines,
    inlineWKReplicas,
  ]);

  // ── Handlers ───────────────────────────────────────────────────────────────
  const handleCopy = () => {
    navigator.clipboard.writeText(generatedYaml).then(() => {
      setCopyLabel('copied');
      setTimeout(() => setCopyLabel('copy'), 2000);
    });
  };

  const handleDownload = () => {
    const blob = new Blob([generatedYaml], { type: 'application/x-yaml' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${name}.yaml`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  const handleApply = () => {
    setApplyState('loading');
    axios
      .post('/api/apply', generatedYaml, { headers: { 'Content-Type': 'application/x-yaml' } })
      .then(() => {
        setApplyState('success');
        onApplySuccess();
        setTimeout(() => setApplyState('idle'), 2000);
      })
      .catch(err => {
        setApplyState('idle');
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
      });
  };

  // ── Form sections ──────────────────────────────────────────────────────────
  const renderTypeFields = () => {
    switch (resourceType) {
      case 'TalosCluster':
        return (
          <>
            <SelectField
              label="Definition Mode"
              value={talosClusterDefinitionMode}
              onChange={setTalosClusterDefinitionMode}
              options={[
                { value: 'inline', label: 'Define Inline' },
                { value: 'reference', label: 'Reference Existing' },
              ]}
            />

            {talosClusterDefinitionMode === 'reference' ? (
              <>
                <Field label="Control Plane Reference Name">
                  <input
                    className={inputCls}
                    type="text"
                    value={controlPlaneRef}
                    onChange={e => setControlPlaneRef(e.target.value)}
                  />
                </Field>
                <Field label="Worker Reference Name">
                  <input
                    className={inputCls}
                    type="text"
                    value={workerRef}
                    onChange={e => setWorkerRef(e.target.value)}
                  />
                </Field>
              </>
            ) : (
              <>
                <SelectField
                  label="Deployment Mode"
                  value={mode}
                  onChange={setMode}
                  options={[
                    { value: 'metal', label: 'Metal (bare-metal)' },
                    { value: 'container', label: 'Container' },
                  ]}
                />

                <Divider label="Control Plane" />

                <Field label="Talos Version">
                  <input
                    className={inputCls}
                    type="text"
                    value={inlineCPTalosVersion}
                    onChange={e => setInlineCPTalosVersion(e.target.value)}
                  />
                </Field>
                <Field label="Kubernetes Version">
                  <input
                    className={inputCls}
                    type="text"
                    value={inlineCPKubernetesVersion}
                    onChange={e => setInlineCPKubernetesVersion(e.target.value)}
                  />
                </Field>

                {mode === 'metal' ? (
                  <>
                    <Field label="Control Plane Endpoint">
                      <input
                        className={inputCls}
                        type="text"
                        value={inlineCPEndpoint}
                        onChange={e => setInlineCPEndpoint(e.target.value)}
                      />
                    </Field>
                    <Field label="Control Plane Machines (one IP per line)">
                      <textarea
                        className={textareaCls}
                        rows={3}
                        value={inlineCPMachines}
                        onChange={e => setInlineCPMachines(e.target.value)}
                      />
                    </Field>
                  </>
                ) : (
                  <Field label="Replicas">
                    <input
                      className={inputCls}
                      type="number"
                      value={inlineCPReplicas}
                      onChange={e => setInlineCPReplicas(parseInt(e.target.value, 10))}
                    />
                  </Field>
                )}

                <Divider label="Worker" />

                <Field label="Talos Version">
                  <input
                    className={inputCls}
                    type="text"
                    value={inlineWKTalosVersion}
                    onChange={e => setInlineWKTalosVersion(e.target.value)}
                  />
                </Field>
                <Field label="Kubernetes Version">
                  <input
                    className={inputCls}
                    type="text"
                    value={inlineWKKubernetesVersion}
                    onChange={e => setInlineWKKubernetesVersion(e.target.value)}
                  />
                </Field>

                {mode === 'metal' ? (
                  <Field label="Worker Machines (one IP per line)">
                    <textarea
                      className={textareaCls}
                      rows={3}
                      value={inlineWKMachines}
                      onChange={e => setInlineWKMachines(e.target.value)}
                    />
                  </Field>
                ) : (
                  <Field label="Replicas">
                    <input
                      className={inputCls}
                      type="number"
                      value={inlineWKReplicas}
                      onChange={e => setInlineWKReplicas(parseInt(e.target.value, 10))}
                    />
                  </Field>
                )}
              </>
            )}
          </>
        );

      case 'TalosControlPlane':
      case 'TalosWorker':
        return (
          <>
            <SelectField
              label="Deployment Mode"
              value={mode}
              onChange={setMode}
              options={[
                { value: 'metal', label: 'Metal (bare-metal)' },
                { value: 'container', label: 'Container' },
              ]}
            />
            <Field label="Talos Version">
              <input
                className={inputCls}
                type="text"
                value={talosVersion}
                onChange={e => setTalosVersion(e.target.value)}
              />
            </Field>
            <Field label="Kubernetes Version">
              <input
                className={inputCls}
                type="text"
                value={kubernetesVersion}
                onChange={e => setKubernetesVersion(e.target.value)}
              />
            </Field>

            {resourceType === 'TalosWorker' && (
              <Field label="Control Plane Reference Name">
                <input
                  className={inputCls}
                  type="text"
                  value={workerControlPlaneRef}
                  onChange={e => setWorkerControlPlaneRef(e.target.value)}
                />
              </Field>
            )}

            {mode === 'metal' ? (
              <>
                {resourceType === 'TalosControlPlane' && (
                  <Field label="Control Plane Endpoint">
                    <input
                      className={inputCls}
                      type="text"
                      value={controlPlaneEndpoint}
                      onChange={e => setControlPlaneEndpoint(e.target.value)}
                    />
                  </Field>
                )}
                <Field label="Machines (one IP per line)">
                  <textarea
                    className={textareaCls}
                    rows={3}
                    value={machines}
                    onChange={e => setMachines(e.target.value)}
                  />
                </Field>
              </>
            ) : (
              <Field label="Replicas">
                <input
                  className={inputCls}
                  type="number"
                  value={replicas}
                  onChange={e => setReplicas(parseInt(e.target.value, 10))}
                />
              </Field>
            )}
          </>
        );

      default:
        return null;
    }
  };

  // ── Render ─────────────────────────────────────────────────────────────────
  return (
    <div className="flex h-full overflow-hidden">
      {/* Left panel – Configuration form */}
      <div className="w-96 flex-shrink-0 border-r border-zinc-800 flex flex-col overflow-hidden">
        <div className="px-5 py-4 border-b border-zinc-800 flex-shrink-0">
          <h2 className="text-sm font-semibold text-zinc-200">Configuration</h2>
          <p className="text-xs text-zinc-500 mt-0.5">Fill in the fields to generate a resource manifest</p>
        </div>

        <div className="flex-1 overflow-y-auto px-5 py-4">
          {/* Resource type */}
          <SelectField
            label="Resource Type"
            value={resourceType}
            onChange={setResourceType}
            options={[
              { value: 'TalosCluster', label: 'TalosCluster' },
              { value: 'TalosControlPlane', label: 'TalosControlPlane' },
              { value: 'TalosWorker', label: 'TalosWorker' },
            ]}
          />

          <Divider label="Metadata" />

          {/* Common fields */}
          <Field label="Name">
            <input
              className={inputCls}
              type="text"
              value={name}
              onChange={e => setName(e.target.value)}
            />
          </Field>
          <Field label="Namespace">
            <input
              className={inputCls}
              type="text"
              value={namespace}
              onChange={e => setNamespace(e.target.value)}
            />
          </Field>

          <Divider label="Spec" />

          {/* Type-specific fields */}
          {renderTypeFields()}
        </div>
      </div>

      {/* Right panel – YAML preview */}
      <div className="flex-1 flex flex-col overflow-hidden">
        <div className="px-5 py-4 border-b border-zinc-800 flex items-center justify-between flex-shrink-0">
          <div>
            <h2 className="text-sm font-semibold text-zinc-200">Generated YAML</h2>
            <p className="text-xs text-zinc-500 mt-0.5">Live preview — updates as you type</p>
          </div>
          <div className="flex items-center gap-2">
            {/* Apply */}
            <button
              onClick={handleApply}
              disabled={applyState === 'loading'}
              className={`flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-all disabled:opacity-50 ${
                applyState === 'success'
                  ? 'bg-green-900 text-green-300 border border-green-800'
                  : 'bg-brand text-white hover:bg-brand-hover'
              }`}
            >
              {applyState === 'success' ? (
                <>
                  <Check size={12} />
                  Applied
                </>
              ) : applyState === 'loading' ? (
                <>
                  <div className="w-3 h-3 border border-white/30 border-t-white rounded-full animate-spin" />
                  Applying…
                </>
              ) : (
                <>
                  <Send size={12} />
                  Apply
                </>
              )}
            </button>

            {/* Copy */}
            <button
              onClick={handleCopy}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium bg-zinc-800 text-zinc-300 hover:bg-zinc-700 hover:text-zinc-100 transition-colors border border-zinc-700"
            >
              {copyLabel === 'copied' ? (
                <>
                  <Check size={12} className="text-green-400" />
                  Copied
                </>
              ) : (
                <>
                  <Copy size={12} />
                  Copy
                </>
              )}
            </button>

            {/* Download */}
            <button
              onClick={handleDownload}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium bg-zinc-800 text-zinc-300 hover:bg-zinc-700 hover:text-zinc-100 transition-colors border border-zinc-700"
            >
              <Download size={12} />
              Download
            </button>
          </div>
        </div>

        {/* YAML content */}
        <div className="flex-1 overflow-auto p-5">
          <pre className="text-xs font-mono text-zinc-300 leading-relaxed whitespace-pre">{generatedYaml}</pre>
        </div>
      </div>
    </div>
  );
}

export default TalosResourceForm;
