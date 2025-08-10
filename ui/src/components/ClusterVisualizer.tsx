import React, { useState, useEffect, useCallback } from 'react';
import ReactFlow, { isNode, Node, Edge, useNodesState, useEdgesState, Controls, Background, BackgroundVariant } from 'reactflow';
import 'reactflow/dist/style.css';
import axios from 'axios';
import { Card, Modal } from 'react-bootstrap';
import YAML from 'js-yaml';
import dagre from 'dagre';

// Dagre graph for layouting
const g = new dagre.graphlib.Graph();
g.setGraph({ rankdir: 'TB' }); // Top-to-bottom layout
g.setDefaultNodeLabel(() => ({}));
g.setDefaultEdgeLabel(() => ({}));

const getLayoutedElements = (nodes: Node[], edges: Edge[]) => {
  g.nodes().forEach((node) => g.removeNode(node));
  g.edges().forEach((edge) => g.removeEdge(edge.v, edge.w));

  nodes.forEach((node) => g.setNode(node.id, { width: 150, height: 50 })); // Set a default width and height for nodes
  edges.forEach((edge) => g.setEdge(edge.source, edge.target));

  dagre.layout(g);

  return nodes.map((node) => {
    const nodeWithPosition = g.node(node.id);
    return {
      ...node,
      position: { x: nodeWithPosition.x - nodeWithPosition.width / 2, y: nodeWithPosition.y - nodeWithPosition.height / 2 },
    };
  });
};

const ClusterVisualizer = () => {
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [showModal, setShowModal] = useState(false);
  const [selectedNode, setSelectedNode] = useState<any>(null);

  const handleClose = () => setShowModal(false);

  const onNodeClick = useCallback((event: React.MouseEvent, node: Node) => {
    setSelectedNode(node.data);
    setShowModal(true);
  }, []);

  useEffect(() => {
    axios.get('/api/resources')
      .then(response => {
        console.log("Backend Response:", response.data);
        const { talosClusters, talosControlPlanes, talosWorkers, talosMachines } = response.data;
        const newNodes: Node[] = [];
        const newEdges: Edge[] = [];

        // Add nodes
        talosClusters.forEach((cluster: any) => {
          newNodes.push({ id: cluster.metadata.name, type: 'default', data: { label: `TalosCluster: ${cluster.metadata.name}`, ...cluster }, position: { x: 0, y: 0 } });
        });

        talosControlPlanes.forEach((cp: any) => {
          newNodes.push({ id: cp.metadata.name, type: 'default', data: { label: `TalosControlPlane: ${cp.metadata.name}`, ...cp }, position: { x: 0, y: 0 } });
        });

        talosWorkers.forEach((worker: any) => {
          newNodes.push({ id: worker.metadata.name, type: 'default', data: { label: `TalosWorker: ${worker.metadata.name}`, ...worker }, position: { x: 0, y: 0 } });
        });

        talosMachines.forEach((machine: any) => {
          newNodes.push({ id: machine.metadata.name, type: 'default', data: { label: `TalosMachine: ${machine.metadata.name}`, ...machine }, position: { x: 0, y: 0 } });
        });

        // Add edges
        talosClusters.forEach((cluster: any) => {
          if (cluster.spec.controlPlaneRef) {
            newEdges.push({ id: `e-${cluster.metadata.name}-${cluster.spec.controlPlaneRef.name}`, source: cluster.metadata.name, target: cluster.spec.controlPlaneRef.name, animated: true });
          }
          if (cluster.spec.workerRef) {
            newEdges.push({ id: `e-${cluster.metadata.name}-${cluster.spec.workerRef.name}`, source: cluster.metadata.name, target: cluster.spec.workerRef.name, animated: true });
          }
        });

        talosWorkers.forEach((worker: any) => {
          if (worker.spec.controlPlaneRef) {
            newEdges.push({ id: `e-${worker.metadata.name}-${worker.spec.controlPlaneRef.name}`, source: worker.metadata.name, target: worker.spec.controlPlaneRef.name, animated: true });
          }
        });

        talosMachines.forEach((machine: any) => {
          if (machine.metadata.ownerReferences && machine.metadata.ownerReferences.length > 0) {
            const owner = machine.metadata.ownerReferences[0].name;
            newEdges.push({ id: `e-${owner}-${machine.metadata.name}`, source: owner, target: machine.metadata.name, animated: true });
          }
        });

        const layoutedNodes = getLayoutedElements(newNodes, newEdges);
        setNodes(layoutedNodes);
        setEdges(newEdges);
      })
      .catch(error => {
        console.error("Error fetching resources:", error);
      });
  }, []);

  return (
    <Card>
      <Card.Body>
        <div style={{ height: '500px', width: '100%' }}>
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onNodeClick={onNodeClick}
            fitView
          >
            <Controls />
            <Background variant={BackgroundVariant.Dots} gap={12} size={1} />
          </ReactFlow>
        </div>
        <Modal show={showModal} onHide={handleClose}>
          <Modal.Header closeButton>
            <Modal.Title>{selectedNode?.metadata.name}</Modal.Title>
          </Modal.Header>
          <Modal.Body style={{ maxHeight: '400px', overflowY: 'auto' }}>
            <pre>
              <code>{YAML.dump(selectedNode)}</code>
            </pre>
          </Modal.Body>
        </Modal>
      </Card.Body>
    </Card>
  );
};

export default ClusterVisualizer;