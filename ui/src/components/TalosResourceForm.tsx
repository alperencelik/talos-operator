import React, { useState, useEffect } from 'react';
import { Container, Row, Col, Form, Button, Card, Tab, Nav, Toast, ToastContainer } from 'react-bootstrap';
import 'bootstrap/dist/css/bootstrap.min.css';
import ClusterVisualizer from './ClusterVisualizer';
import 'reactflow/dist/style.css';
import YAML from 'js-yaml';
import axios from 'axios';

// --- Main App Component ---
function TalosResourceForm() {
  // --- State ---
  const [resourceType, setResourceType] = useState('TalosCluster');

  // Common
  const [name, setName] = useState('sample');
  const [namespace, setNamespace] = useState('default');
  const [mode, setMode] = useState('metal');
  const [talosVersion, setTalosVersion] = useState('v1.10.3');
  const [kubernetesVersion, setKubernetesVersion] = useState('v1.31.0');
  const [replicas, setReplicas] = useState(2);
  const [machines, setMachines] = useState('<machine-ip-1>\n<machine-ip-2>');

  // TalosCluster specific
  const [talosClusterDefinitionMode, setTalosClusterDefinitionMode] = useState('inline'); // 'inline' or 'reference'
  const [controlPlaneRef, setControlPlaneRef] = useState('taloscontrolplane-sample');
  const [workerRef, setWorkerRef] = useState('talosworker-sample');

  // Inline Control Plane for TalosCluster
  const [inlineCPTalosVersion, setInlineCPTalosVersion] = useState('v1.10.3');
  const [inlineCPKubernetesVersion, setInlineCPKubernetesVersion] = useState('v1.31.0');
  const [inlineCPEndpoint, setInlineCPEndpoint] = useState('https://<control-plane-endpoint>:6443');
  const [inlineCPMachines, setInlineCPMachines] = useState('<control-plane-machine-ip-1>\n<control-plane-machine-ip-2>');
  const [inlineCPReplicas, setInlineCPReplicas] = useState(2);

  // Inline Worker for TalosCluster
  const [inlineWKTalosVersion, setInlineWKTalosVersion] = useState('v1.10.3');
  const [inlineWKKubernetesVersion, setInlineWKKubernetesVersion] = useState('v1.31.0');
  const [inlineWKMachines, setInlineWKMachines] = useState('<worker-machine-ip-1>\n<worker-machine-ip-2>');
  const [inlineWKReplicas, setInlineWKReplicas] = useState(2);


  // TalosControlPlane specific
  const [controlPlaneEndpoint, setControlPlaneEndpoint] = useState('https://<control-plane-endpoint>:6443');

  // TalosWorker specific
  const [workerControlPlaneRef, setWorkerControlPlaneRef] = useState('taloscontrolplane-sample');


  const [generatedYaml, setGeneratedYaml] = useState('');
  const [copySuccess, setCopySuccess] = useState('');
  const [applySuccess, setApplySuccess] = useState('');
  const [clusterResources, setClusterResources] = useState<any>(null);

  type Notice = { variant: 'success' | 'danger' | 'warning' | 'info'; message: string };
  const [notice, setNotice] = useState<Notice | null>(null);
  const [activeTab, setActiveTab] = useState<'generator' | 'visualizer'>(() => {
    try {
      const saved = typeof window !== 'undefined' ? window.localStorage.getItem('activeTab') : null;
      return saved === 'visualizer' || saved === 'generator' ? (saved as 'generator' | 'visualizer') : 'generator';
    } catch {
      return 'generator';
    }
  });
  const [resourcesStale, setResourcesStale] = useState<boolean>(false);

  // Extract a human-friendly message from Axios errors (network / response / other)
  const extractAxiosError = (err: any): string => {
    // Use axios.isAxiosError to discriminate
    if (axios.isAxiosError && axios.isAxiosError(err)) {
      if (err.response) {
        const status = err.response.status;
        const statusText = (err.response as any).statusText || '';
        const data = typeof err.response.data === 'string' ? err.response.data : JSON.stringify(err.response.data);
        return `status ${status}${statusText ? ` ${statusText}` : ''}${data ? `: ${data}` : ''}`;
      }
      if (err.request) {
        return 'No response received from the server (network error or CORS).';
      }
      return err.message || 'Request failed.';
    }
    try { return (err && (err as any).message) ? (err as any).message : String(err); } catch { return 'Unknown error'; }
  };

//


  // --- Handlers ---
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

  const handleCopy = () => {
    navigator.clipboard.writeText(generatedYaml).then(() => {
      setCopySuccess('Copied!');
      setTimeout(() => setCopySuccess(''), 2000);
    }, () => {
      setCopySuccess('Failed to copy!');
      setTimeout(() => setCopySuccess(''), 2000);
    });
  };

  const handleApply = () => {
    axios.post('/api/apply', generatedYaml, { headers: { 'Content-Type': 'application/x-yaml' } })
      .then(() => {
        setApplySuccess('Applied!');
        setTimeout(() => setApplySuccess(''), 2000);
        setResourcesStale(true);
        if (activeTab === 'visualizer') {
          fetchResources();
          setResourcesStale(false);
        }
      })
      .catch(err => {
        const msg = extractAxiosError(err);
        setApplySuccess(`Apply failed: ${msg}`);
        setNotice({ variant: 'danger', message: `Apply failed: ${msg}` });
        setTimeout(() => setApplySuccess(''), 5000);
      });
  };

  const fetchResources = () => {
    axios.get('/api/resources')
      .then(response => {
        setClusterResources(response.data);
      })
      .catch(error => {
        setClusterResources(null);
        const msg = extractAxiosError(error);
        setNotice({ variant: 'danger', message: `Failed to fetch cluster resources: ${msg}` });
        console.error('Error fetching resources:', error);
      });
  };

  // --- Effects ---

  // (Effect to clear errors for fields that are not currently visible removed)

  useEffect(() => {
    // Reset state on resource type change
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
    setTalosClusterDefinitionMode('inline'); // Default to inline
    setInlineCPTalosVersion('v1.10.3');
    setInlineCPKubernetesVersion('v1.31.0');
    setInlineCPEndpoint('https://<control-plane-endpoint>:6443');
    setInlineCPMachines('<control-plane-machine-ip-1>\n<control-plane-machine-ip-2>');
    setInlineCPReplicas(2);
    setInlineWKTalosVersion('v1.10.3');
    setInlineWKKubernetesVersion('v1.31.0');
    setInlineWKMachines('<worker-machine-ip-1>\n<worker-machine-ip-2>');
    setInlineWKReplicas(2);
    // setErrors({}); // removed
  }, [resourceType]);

  useEffect(() => {
    let resource: any;
    const apiVersion = 'talos.alperen.cloud/v1alpha1';

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
        } else { // inline
          const inlineCPSpec: any = {
            version: inlineCPTalosVersion,
            kubeVersion: inlineCPKubernetesVersion,
            mode: mode,
          };
          if (mode === 'metal') {
            inlineCPSpec.endpoint = inlineCPEndpoint;
            inlineCPSpec.metalSpec = { machines: inlineCPMachines.split('\n').filter(m => m.trim() !== '') };
          } else {
            inlineCPSpec.replicas = inlineCPReplicas;
          }

          const inlineWKSpec: any = {
            version: inlineWKTalosVersion,
            kubeVersion: inlineWKKubernetesVersion,
            mode: mode,
            controlPlaneRef: { name: `${name}-controlplane` }, // Derived from cluster name
          };
          if (mode === 'metal') {
            inlineWKSpec.metalSpec = { machines: inlineWKMachines.split('\n').filter(m => m.trim() !== '') };
          } else {
            inlineWKSpec.replicas = inlineWKReplicas;
          }

          resource = {
            apiVersion,
            kind: 'TalosCluster',
            metadata: { name, namespace },
            spec: {
              controlPlane: inlineCPSpec,
              worker: inlineWKSpec,
            },
          };
        }
        break;
      case 'TalosControlPlane':
        const cpSpec: any = {
            version: talosVersion,
            kubeVersion: kubernetesVersion,
            mode: mode,
        };
        if (mode === 'metal') {
            cpSpec.endpoint = controlPlaneEndpoint;
            cpSpec.metalSpec = { machines: machines.split('\n').filter(m => m.trim() !== '') };
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
      case 'TalosWorker':
        const workerSpec: any = {
            version: talosVersion,
            kubeVersion: kubernetesVersion,
            mode: mode,
            controlPlaneRef: { name: workerControlPlaneRef }
        };
        if (mode === 'metal') {
            workerSpec.metalSpec = { machines: machines.split('\n').filter(m => m.trim() !== '') };
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
      default:
        resource = { error: 'Invalid resource type selected' };
    }

    setGeneratedYaml(YAML.dump(resource));
  }, [
    resourceType, name, namespace, mode, talosVersion, kubernetesVersion, machines, replicas,
    controlPlaneEndpoint, controlPlaneRef, workerRef, workerControlPlaneRef,
    talosClusterDefinitionMode, inlineCPTalosVersion, inlineCPKubernetesVersion, inlineCPEndpoint, inlineCPMachines, inlineCPReplicas,
    inlineWKTalosVersion, inlineWKKubernetesVersion, inlineWKMachines, inlineWKReplicas
  ]);

  // (Field onChange wrappers with live validation removed)

  // --- Render ---
  const renderForm = () => {
    switch (resourceType) {
      case 'TalosCluster':
        return (
          <>
            <Form.Group className="mb-3">
              <Form.Label>Definition Mode</Form.Label>
              <Form.Select value={talosClusterDefinitionMode} onChange={e => setTalosClusterDefinitionMode(e.target.value)}>
                <option value="inline">Define Inline</option>
                <option value="reference">Reference Existing</option>
              </Form.Select>
            </Form.Group>
            {talosClusterDefinitionMode === 'reference' ? (
              <>
                <Form.Group className="mb-3">
                  <Form.Label>Control Plane Reference Name</Form.Label>
                  <Form.Control type="text" value={controlPlaneRef} onChange={e => setControlPlaneRef(e.target.value)} />
                </Form.Group>
                <Form.Group className="mb-3">
                  <Form.Label>Worker Reference Name</Form.Label>
                  <Form.Control type="text" value={workerRef} onChange={e => setWorkerRef(e.target.value)} />
                </Form.Group>
              </>
            ) : (
              <>
                <Form.Group className="mb-3">
                  <Form.Label>Deployment Mode</Form.Label>
                  <Form.Select value={mode} onChange={e => setMode(e.target.value)}>
                    <option value="metal">Metal</option>
                    <option value="container">Container</option>
                  </Form.Select>
                </Form.Group>
                <hr />
                <h5>Control Plane (Inline)</h5>
                <Form.Group className="mb-3">
                  <Form.Label>Talos Version</Form.Label>
                  <Form.Control type="text" value={inlineCPTalosVersion} onChange={e => setInlineCPTalosVersion(e.target.value)} />
                </Form.Group>
                <Form.Group className="mb-3">
                  <Form.Label>Kubernetes Version</Form.Label>
                  <Form.Control type="text" value={inlineCPKubernetesVersion} onChange={e => setInlineCPKubernetesVersion(e.target.value)} />
                </Form.Group>
                {mode === 'metal' ? (
                  <>
                    <Form.Group className="mb-3">
                      <Form.Label>Control Plane Endpoint</Form.Label>
                      <Form.Control type="text" value={inlineCPEndpoint} onChange={e => setInlineCPEndpoint(e.target.value)} />
                    </Form.Group>
                    <Form.Group className="mb-3">
                      <Form.Label>Control Plane Machines (one IP per line)</Form.Label>
                      <Form.Control as="textarea" rows={3} value={inlineCPMachines} onChange={e => setInlineCPMachines(e.target.value)} />
                    </Form.Group>
                  </>
                ) : (
                  <Form.Group className="mb-3">
                    <Form.Label>Replicas</Form.Label>
                    <Form.Control type="number" value={inlineCPReplicas} onChange={e => setInlineCPReplicas(parseInt(e.target.value, 10))} />
                  </Form.Group>
                )}
                <hr />
                <h5>Worker (Inline)</h5>
                <Form.Group className="mb-3">
                  <Form.Label>Talos Version</Form.Label>
                  <Form.Control type="text" value={inlineWKTalosVersion} onChange={e => setInlineWKTalosVersion(e.target.value)} />
                </Form.Group>
                <Form.Group className="mb-3">
                  <Form.Label>Kubernetes Version</Form.Label>
                  <Form.Control type="text" value={inlineWKKubernetesVersion} onChange={e => setInlineWKKubernetesVersion(e.target.value)} />
                </Form.Group>
                {mode === 'metal' ? (
                  <Form.Group className="mb-3">
                    <Form.Label>Worker Machines (one IP per line)</Form.Label>
                    <Form.Control as="textarea" rows={3} value={inlineWKMachines} onChange={e => setInlineWKMachines(e.target.value)} />
                  </Form.Group>
                ) : (
                  <Form.Group className="mb-3">
                    <Form.Label>Replicas</Form.Label>
                    <Form.Control type="number" value={inlineWKReplicas} onChange={e => setInlineWKReplicas(parseInt(e.target.value, 10))} />
                  </Form.Group>
                )}
              </>
            )}
          </>
        );
      case 'TalosControlPlane':
      case 'TalosWorker':
        return (
          <>
            <Form.Group className="mb-3">
              <Form.Label>Deployment Mode</Form.Label>
              <Form.Select value={mode} onChange={e => setMode(e.target.value)}>
                <option value="metal">Metal</option>
                <option value="container">Container</option>
              </Form.Select>
            </Form.Group>
            <Form.Group className="mb-3">
              <Form.Label>Talos Version</Form.Label>
              <Form.Control type="text" value={talosVersion} onChange={e => setTalosVersion(e.target.value)} />
            </Form.Group>
            <Form.Group className="mb-3">
              <Form.Label>Kubernetes Version</Form.Label>
              <Form.Control type="text" value={kubernetesVersion} onChange={e => setKubernetesVersion(e.target.value)} />
            </Form.Group>
            {resourceType === 'TalosWorker' && (
                 <Form.Group className="mb-3">
                    <Form.Label>Control Plane Reference Name</Form.Label>
                    <Form.Control type="text" value={workerControlPlaneRef} onChange={e => setWorkerControlPlaneRef(e.target.value)} />
                 </Form.Group>
            )}
            {mode === 'metal' ? (
              <>
                {resourceType === 'TalosControlPlane' && (
                  <Form.Group className="mb-3">
                    <Form.Label>Control Plane Endpoint</Form.Label>
                    <Form.Control type="text" value={controlPlaneEndpoint} onChange={e => setControlPlaneEndpoint(e.target.value)} />
                  </Form.Group>
                )}
                <Form.Group className="mb-3">
                  <Form.Label>Machines (one IP per line)</Form.Label>
                  <Form.Control as="textarea" rows={3} value={machines} onChange={e => setMachines(e.target.value)} />
                </Form.Group>
              </>
            ) : (
              <Form.Group className="mb-3">
                <Form.Label>Replicas</Form.Label>
                <Form.Control type="number" value={replicas} onChange={e => setReplicas(parseInt(e.target.value, 10))} />
              </Form.Group>
            )}
          </>
        );
      default:
        return null;
    }
  };

  const renderResources = (title: string, resources: any[]) => (
    <Card className="mb-3">
      <Card.Body>
        <Card.Title>{title}</Card.Title>
        {resources && resources.length > 0 ? (
          <pre style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>
            <code>{YAML.dump(resources)}</code>
          </pre>
        ) : (
          <p>No resources found.</p>
        )}
      </Card.Body>
    </Card>
  );

  return (
    <Container fluid className="p-4">
      <Row>
        <Col>
          <h1 className="mb-4">Talos Operator UI</h1>
        </Col>
      </Row>
      <ToastContainer position="bottom-end" className="p-3">
        {notice && (
          <Toast
            onClose={() => setNotice(null)}
            show={!!notice}
            delay={5000}
            autohide
            bg={notice.variant === 'warning' || notice.variant === 'info' ? 'light' : notice.variant}
            className={notice.variant === 'warning' || notice.variant === 'info' ? '' : 'text-white'}
          >
            <Toast.Header closeButton={false}>
              <strong className="me-auto">
                {notice.variant === 'danger' ? 'Error' : notice.variant.charAt(0).toUpperCase() + notice.variant.slice(1)}
              </strong>
            </Toast.Header>
            <Toast.Body>{notice.message}</Toast.Body>
          </Toast>
        )}
      </ToastContainer>
      <Tab.Container
        activeKey={activeTab}
        onSelect={(k) => {
          const key = (k ?? 'generator') as 'generator' | 'visualizer';
          setActiveTab(key);
          try { window.localStorage.setItem('activeTab', key); } catch {}
          if (key === 'visualizer' && (resourcesStale || !clusterResources)) {
            fetchResources();
            setResourcesStale(false);
          }
        }}
      >
        <Row>
          <Col sm={3}>
            <Nav variant="pills" className="flex-column">
              <Nav.Item>
                <Nav.Link eventKey="generator">Resource Generator</Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link eventKey="visualizer">Resource Visualizer</Nav.Link>
              </Nav.Item>
            </Nav>
          </Col>
          <Col sm={9}>
            <Tab.Content>
              <Tab.Pane eventKey="generator">
                <Row>
                  <Col md={6}>
                    <Card>
                      <Card.Body>
                        <Card.Title>Resource Configuration</Card.Title>
                        <Form>
                          <Form.Group className="mb-3">
                            <Form.Label>Resource Type</Form.Label>
                            <Form.Select value={resourceType} onChange={e => setResourceType(e.target.value)}>
                              <option value="TalosCluster">TalosCluster</option>
                              <option value="TalosControlPlane">TalosControlPlane</option>
                              <option value="TalosWorker">TalosWorker</option>
                            </Form.Select>
                          </Form.Group>
                          <hr/>
                          <Form.Group className="mb-3">
                            <Form.Label>Name</Form.Label>
                            <Form.Control type="text" value={name} onChange={e => setName(e.target.value)} />
                          </Form.Group>
                          <Form.Group className="mb-3">
                            <Form.Label>Namespace</Form.Label>
                            <Form.Control type="text" value={namespace} onChange={e => setNamespace(e.target.value)} />
                          </Form.Group>
                          {renderForm()}
                        </Form>
                      </Card.Body>
                    </Card>
                  </Col>
                  <Col md={6}>
                    <Card>
                      <Card.Body>
                        <Card.Title>Generated YAML</Card.Title>
                        <pre style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-all', backgroundColor: '#f8f9fa', padding: '1rem', borderRadius: '0.25rem' }}>
                          <code>{generatedYaml}</code>
                        </pre>
                        <Button variant="success" onClick={handleApply}>{applySuccess || 'Apply'}</Button>
                        <Button variant="secondary" onClick={handleCopy} className="ms-2">{copySuccess || 'Copy YAML'}</Button>
                        <Button variant="primary" onClick={handleDownload} className="ms-2">Download YAML</Button>
                      </Card.Body>
                    </Card>
                  </Col>
                </Row>
              </Tab.Pane>
              <Tab.Pane eventKey="visualizer">
                <ClusterVisualizer />
              </Tab.Pane>
            </Tab.Content>
          </Col>
        </Row>
      </Tab.Container>
    </Container>
  );
}

export default TalosResourceForm;
