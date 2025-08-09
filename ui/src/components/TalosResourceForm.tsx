import React, { useState, useEffect } from 'react';
import { Container, Row, Col, Form, Button, Card, Tab, Nav } from 'react-bootstrap';
import 'bootstrap/dist/css/bootstrap.min.css';
import YAML from 'js-yaml';
import axios from 'axios';

// --- Validation Functions ---
function isRFC1123(name: string): boolean {
  const rfc1123Regex = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/;
  return rfc1123Regex.test(name);
}

function isValidIPv4(ip: string): boolean {
  const ipv4Regex = /^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$/;
  return ipv4Regex.test(ip);
}

function isValidURLWithIPv4(url: string): boolean {
  const urlRegex = /^https?:\/\/(?:[0-9]{1,3}\.){3}[0-9]{1,3}:[0-9]+$/;
  return urlRegex.test(url);
}

// --- Main App Component ---
function TalosResourceForm() {
  // --- State ---
  const [resourceType, setResourceType] = useState('TalosCluster');

  // Common
  const [name, setName] = useState('sample');
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
  const [errors, setErrors] = useState<any>({});
  const [touched, setTouched] = useState<any>({});
  const [copySuccess, setCopySuccess] = useState('');
  const [applySuccess, setApplySuccess] = useState('');
  const [clusterResources, setClusterResources] = useState<any>(null);

  // --- Handlers ---
  const handleBlur = (field: string) => {
    setTouched({ ...touched, [field]: true });
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
        fetchResources();
      })
      .catch(err => {
        setApplySuccess(`Apply failed: ${err.response?.data || err.message}`);
        setTimeout(() => setApplySuccess(''), 5000);
      });
  };

  const fetchResources = () => {
    axios.get('/api/resources')
      .then(response => {
        setClusterResources(response.data);
      })
      .catch(error => {
        console.error("Error fetching resources:", error);
      });
  };

  // --- Effects ---
  useEffect(() => {
    fetchResources();
  }, []);

  useEffect(() => {
    // Reset state on resource type change
    setName('sample');
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
    setErrors({});
    setTouched({});
  }, [resourceType]);

  useEffect(() => {
    const newErrors: any = {};

    if (touched.name && !name) {
        newErrors.name = 'Name is required.';
    } else if (touched.name && !isRFC1123(name)) {
        newErrors.name = 'Name must be RFC1123 compliant.';
    } else if (touched.name && name.length > 63) {
        newErrors.name = 'Name cannot be longer than 63 characters.';
    }

    // Validation for TalosControlPlane and TalosWorker (standalone and inline)
    const validateVersionFields = (version: string, kubeVersion: string, prefix: string) => {
      if (touched[`${prefix}TalosVersion`] && !version) newErrors[`${prefix}TalosVersion`] = `${prefix}Talos version is required.`;
      if (touched[`${prefix}KubernetesVersion`] && !kubeVersion) newErrors[`${prefix}KubernetesVersion`] = `${prefix}Kubernetes version is required.`;
    };

    const validateMetalFields = (endpoint: string, machines: string, endpointPrefix: string, machinesPrefix: string) => {
      if (touched[endpointPrefix] && !isValidURLWithIPv4(endpoint)) {
        newErrors[endpointPrefix] = 'Endpoint must be a valid URL with an IPv4.';
      }
      const machineIPs = machines.split('\n').filter(m => m.trim() !== '');
      if (touched[machinesPrefix] && machineIPs.some(m => !isValidIPv4(m))) {
        newErrors[machinesPrefix] = 'All machines must be valid IPv4 addresses.';
      }
    };

    if (resourceType === 'TalosControlPlane' || resourceType === 'TalosWorker') {
      validateVersionFields(talosVersion, kubernetesVersion, '');
      if (mode === 'metal') {
        validateMetalFields(controlPlaneEndpoint, machines, 'controlPlaneEndpoint', 'machines');
      }
    } else if (resourceType === 'TalosCluster' && talosClusterDefinitionMode === 'inline') {
      validateVersionFields(inlineCPTalosVersion, inlineCPKubernetesVersion, 'inlineCP');
      validateVersionFields(inlineWKTalosVersion, inlineWKKubernetesVersion, 'inlineWK');
      if (mode === 'metal') {
        validateMetalFields(inlineCPEndpoint, inlineCPMachines, 'inlineCPEndpoint', 'inlineCPMachines');
        validateMetalFields('', inlineWKMachines, '', 'inlineWKMachines'); // Worker doesn't have endpoint
      }
    }
    
    if (resourceType === 'TalosCluster' && talosClusterDefinitionMode === 'reference') {
        if (touched.controlPlaneRef && !controlPlaneRef) newErrors.controlPlaneRef = 'Control Plane reference is required.';
        if (touched.workerRef && !workerRef) newErrors.workerRef = 'Worker reference is required.';
    }
    
    if (resourceType === 'TalosWorker') {
        if (touched.workerControlPlaneRef && !workerControlPlaneRef) newErrors.workerControlPlaneRef = 'Control Plane reference is required.';
    }

    setErrors(newErrors);
  }, [
    resourceType, name, mode, talosVersion, kubernetesVersion, machines, 
    controlPlaneEndpoint, controlPlaneRef, workerRef, workerControlPlaneRef, 
    talosClusterDefinitionMode, inlineCPTalosVersion, inlineCPKubernetesVersion, inlineCPEndpoint, inlineCPMachines, inlineCPReplicas,
    inlineWKTalosVersion, inlineWKKubernetesVersion, inlineWKMachines, inlineWKReplicas, touched
  ]);

  useEffect(() => {
    let resource: any;
    const apiVersion = 'talos.alperen.cloud/v1alpha1';

    switch (resourceType) {
      case 'TalosCluster':
        if (talosClusterDefinitionMode === 'reference') {
          resource = {
            apiVersion,
            kind: 'TalosCluster',
            metadata: { name },
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
            metadata: { name },
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
          metadata: { name },
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
          metadata: { name },
          spec: workerSpec,
        };
        break;
      default:
        resource = { error: 'Invalid resource type selected' };
    }

    setGeneratedYaml(YAML.dump(resource));
  }, [
    resourceType, name, mode, talosVersion, kubernetesVersion, machines, replicas,
    controlPlaneEndpoint, controlPlaneRef, workerRef, workerControlPlaneRef,
    talosClusterDefinitionMode, inlineCPTalosVersion, inlineCPKubernetesVersion, inlineCPEndpoint, inlineCPMachines, inlineCPReplicas,
    inlineWKTalosVersion, inlineWKKubernetesVersion, inlineWKMachines, inlineWKReplicas
  ]);

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
                  <Form.Control type="text" value={controlPlaneRef} onChange={e => setControlPlaneRef(e.target.value)} onBlur={() => handleBlur('controlPlaneRef')} isInvalid={!!errors.controlPlaneRef} />
                  <Form.Control.Feedback type="invalid">{errors.controlPlaneRef}</Form.Control.Feedback>
                </Form.Group>
                <Form.Group className="mb-3">
                  <Form.Label>Worker Reference Name</Form.Label>
                  <Form.Control type="text" value={workerRef} onChange={e => setWorkerRef(e.target.value)} onBlur={() => handleBlur('workerRef')} isInvalid={!!errors.workerRef} />
                  <Form.Control.Feedback type="invalid">{errors.workerRef}</Form.Control.Feedback>
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
                  <Form.Control type="text" value={inlineCPTalosVersion} onChange={e => setInlineCPTalosVersion(e.target.value)} onBlur={() => handleBlur('inlineCPTalosVersion')} isInvalid={!!errors.inlineCPTalosVersion} />
                  <Form.Control.Feedback type="invalid">{errors.inlineCPTalosVersion}</Form.Control.Feedback>
                </Form.Group>
                <Form.Group className="mb-3">
                  <Form.Label>Kubernetes Version</Form.Label>
                  <Form.Control type="text" value={inlineCPKubernetesVersion} onChange={e => setInlineCPKubernetesVersion(e.target.value)} onBlur={() => handleBlur('inlineCPKubernetesVersion')} isInvalid={!!errors.inlineCPKubernetesVersion} />
                  <Form.Control.Feedback type="invalid">{errors.inlineCPKubernetesVersion}</Form.Control.Feedback>
                </Form.Group>
                {mode === 'metal' ? (
                  <>
                    <Form.Group className="mb-3">
                      <Form.Label>Control Plane Endpoint</Form.Label>
                      <Form.Control type="text" value={inlineCPEndpoint} onChange={e => setInlineCPEndpoint(e.target.value)} onBlur={() => handleBlur('inlineCPEndpoint')} isInvalid={!!errors.inlineCPEndpoint} />
                      <Form.Control.Feedback type="invalid">{errors.inlineCPEndpoint}</Form.Control.Feedback>
                    </Form.Group>
                    <Form.Group className="mb-3">
                      <Form.Label>Control Plane Machines (one IP per line)</Form.Label>
                      <Form.Control as="textarea" rows={3} value={inlineCPMachines} onChange={e => setInlineCPMachines(e.target.value)} onBlur={() => handleBlur('inlineCPMachines')} isInvalid={!!errors.inlineCPMachines} />
                      <Form.Control.Feedback type="invalid">{errors.inlineCPMachines}</Form.Control.Feedback>
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
                  <Form.Control type="text" value={inlineWKTalosVersion} onChange={e => setInlineWKTalosVersion(e.target.value)} onBlur={() => handleBlur('inlineWKTalosVersion')} isInvalid={!!errors.inlineWKTalosVersion} />
                  <Form.Control.Feedback type="invalid">{errors.inlineWKTalosVersion}</Form.Control.Feedback>
                </Form.Group>
                <Form.Group className="mb-3">
                  <Form.Label>Kubernetes Version</Form.Label>
                  <Form.Control type="text" value={inlineWKKubernetesVersion} onChange={e => setInlineWKKubernetesVersion(e.target.value)} onBlur={() => handleBlur('inlineWKKubernetesVersion')} isInvalid={!!errors.inlineWKKubernetesVersion} />
                  <Form.Control.Feedback type="invalid">{errors.inlineWKKubernetesVersion}</Form.Control.Feedback>
                </Form.Group>
                {mode === 'metal' ? (
                  <Form.Group className="mb-3">
                    <Form.Label>Worker Machines (one IP per line)</Form.Label>
                    <Form.Control as="textarea" rows={3} value={inlineWKMachines} onChange={e => setInlineWKMachines(e.target.value)} onBlur={() => handleBlur('inlineWKMachines')} isInvalid={!!errors.inlineWKMachines} />
                    <Form.Control.Feedback type="invalid">{errors.inlineWKMachines}</Form.Control.Feedback>
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
              <Form.Control type="text" value={talosVersion} onChange={e => setTalosVersion(e.target.value)} onBlur={() => handleBlur('talosVersion')} isInvalid={!!errors.talosVersion} />
              <Form.Control.Feedback type="invalid">{errors.talosVersion}</Form.Control.Feedback>
            </Form.Group>
            <Form.Group className="mb-3">
              <Form.Label>Kubernetes Version</Form.Label>
              <Form.Control type="text" value={kubernetesVersion} onChange={e => setKubernetesVersion(e.target.value)} onBlur={() => handleBlur('kubernetesVersion')} isInvalid={!!errors.kubernetesVersion} />
              <Form.Control.Feedback type="invalid">{errors.kubernetesVersion}</Form.Control.Feedback>
            </Form.Group>
            {resourceType === 'TalosWorker' && (
                 <Form.Group className="mb-3">
                    <Form.Label>Control Plane Reference Name</Form.Label>
                    <Form.Control type="text" value={workerControlPlaneRef} onChange={e => setWorkerControlPlaneRef(e.target.value)} onBlur={() => handleBlur('workerControlPlaneRef')} isInvalid={!!errors.workerControlPlaneRef} />
                    <Form.Control.Feedback type="invalid">{errors.workerControlPlaneRef}</Form.Control.Feedback>
                 </Form.Group>
            )}
            {mode === 'metal' ? (
              <>
                {resourceType === 'TalosControlPlane' && (
                  <Form.Group className="mb-3">
                    <Form.Label>Control Plane Endpoint</Form.Label>
                    <Form.Control type="text" value={controlPlaneEndpoint} onChange={e => setControlPlaneEndpoint(e.target.value)} onBlur={() => handleBlur('controlPlaneEndpoint')} isInvalid={!!errors.controlPlaneEndpoint} />
                    <Form.Control.Feedback type="invalid">{errors.controlPlaneEndpoint}</Form.Control.Feedback>
                  </Form.Group>
                )}
                <Form.Group className="mb-3">
                  <Form.Label>Machines (one IP per line)</Form.Label>
                  <Form.Control as="textarea" rows={3} value={machines} onChange={e => setMachines(e.target.value)} onBlur={() => handleBlur('machines')} isInvalid={!!errors.machines} />
                  <Form.Control.Feedback type="invalid">{errors.machines}</Form.Control.Feedback>
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
          <h1 className="mb-4">Talos Operator UI Helper</h1>
        </Col>
      </Row>
      <Tab.Container defaultActiveKey="generator">
        <Row>
          <Col sm={3}>
            <Nav variant="pills" className="flex-column">
              <Nav.Item>
                <Nav.Link eventKey="generator">Resource Generator</Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link eventKey="visualizer">Cluster Visualizer</Nav.Link>
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
                            <Form.Control type="text" value={name} onChange={e => setName(e.target.value)} onBlur={() => handleBlur('name')} isInvalid={!!errors.name} />
                            <Form.Control.Feedback type="invalid">{errors.name}</Form.Control.Feedback>
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
                        <Button variant="success" onClick={handleApply} disabled={Object.keys(errors).length > 0}>{applySuccess || 'Apply'}</Button>
                        <Button variant="secondary" onClick={handleCopy} disabled={Object.keys(errors).length > 0} className="ms-2">{copySuccess || 'Copy YAML'}</Button>
                        <Button variant="primary" onClick={handleDownload} disabled={Object.keys(errors).length > 0} className="ms-2">Download YAML</Button>
                      </Card.Body>
                    </Card>
                  </Col>
                </Row>
              </Tab.Pane>
              <Tab.Pane eventKey="visualizer">
                {clusterResources ? (
                  <>
                    {renderResources('TalosClusters', clusterResources.talosClusters)}
                    {renderResources('TalosControlPlanes', clusterResources.talosControlPlanes)}
                    {renderResources('TalosWorkers', clusterResources.talosWorkers)}
                  </>
                ) : <p>Loading resources...</p>}
              </Tab.Pane>
            </Tab.Content>
          </Col>
        </Row>
      </Tab.Container>
    </Container>
  );
}

export default TalosResourceForm;