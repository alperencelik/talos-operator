import React, { useState, useEffect, useCallback } from 'react';
import ReactFlow, { Node, Edge, useNodesState, useEdgesState, Controls, Background, BackgroundVariant } from 'reactflow';
import 'reactflow/dist/style.css';
import axios from 'axios';
import { Card, Modal } from 'react-bootstrap';
import YAML from 'js-yaml';
import dagre from 'dagre';
import '../TalosUI.css';

// Dagre graph for layouting
const g = new dagre.graphlib.Graph();
g.setGraph({ rankdir: 'TB' }); // Top-to-bottom layout
g.setDefaultNodeLabel(() => ({}));
g.setDefaultEdgeLabel(() => ({}));

const getLayoutedElements = (nodes: Node[], edges: Edge[]) => {
  g.nodes().forEach((node) => g.removeNode(node));
  g.edges().forEach((edge) => g.removeEdge(edge.v, edge.w));

  nodes.forEach((node) => {
    const nodeData = { width: 250, height: 100 };
    g.setNode(node.id, nodeData);
  });
  edges.forEach((edge) => g.setEdge(edge.source, edge.target));

  dagre.layout(g);

  const layoutedNodes = nodes.map((node) => {
    const nodeWithPosition = g.node(node.id);
    return {
      ...node,
      position: { x: nodeWithPosition.x - nodeWithPosition.width / 2, y: nodeWithPosition.y - nodeWithPosition.height / 2 },
    };
  });

  // Targeted adjustment: Align TalosWorker with its referenced TalosControlPlane,
  // and adjust TalosMachine owned by these workers.
  layoutedNodes.forEach(node => {
    if (node.data.label.startsWith('TalosWorker') && node.data.spec?.controlPlaneRef?.name) {
      const referencedControlPlaneName = node.data.spec.controlPlaneRef.name;
      const controlPlaneNode = layoutedNodes.find(cpNode =>
        cpNode.data.label.startsWith('TalosControlPlane') && cpNode.id === referencedControlPlaneName
      );

      if (controlPlaneNode) {
        // Align worker with control plane
        node.position.y = controlPlaneNode.position.y;

        // Adjust TalosMachine nodes owned by this worker
        layoutedNodes.forEach(machineNode => {
          if (machineNode.data.label.startsWith('TalosMachine') && machineNode.data.metadata?.ownerReferences) {
            const ownerRef = machineNode.data.metadata.ownerReferences.find(
              (ref: any) => ref.kind === 'TalosWorker' && ref.name === node.id
            );
            if (ownerRef) {
              // Position machine below the worker
              machineNode.position.y = node.position.y + (node.height || 100) + 50; // Worker height + some offset
            }
          }
        });
      }
    }
  });

  return layoutedNodes;
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

  const onNodeDrag = useCallback((event: any, draggedNode: Node) => {
    setNodes((nds) => {
      const updatedNodes = nds.map((node) => {
        if (node.id === draggedNode.id) {
          let newX = draggedNode.position.x;
          let newY = draggedNode.position.y;

          const draggedNodeWidth = node.width || 250;
          const draggedNodeHeight = node.height || 100;

          nds.forEach((otherNode) => {
            if (otherNode.id !== draggedNode.id) {
              const otherNodeWidth = otherNode.width || 250;
              const otherNodeHeight = otherNode.height || 100;

              const otherNodePosition = otherNode.position || { x: 0, y: 0 };

              const draggedRect = {
                left: newX,
                right: newX + draggedNodeWidth,
                top: newY,
                bottom: newY + draggedNodeHeight,
              };
              const otherRect = {
                left: otherNodePosition.x,
                right: otherNodePosition.x + otherNodeWidth,
                top: otherNodePosition.y,
                bottom: otherNodePosition.y + otherNodeHeight,
              };

              if (
                draggedRect.left < otherRect.right &&
                draggedRect.right > otherRect.left &&
                draggedRect.top < otherRect.bottom &&
                draggedRect.bottom > otherRect.top
              ) {
                const overlapX = Math.min(draggedRect.right, otherRect.right) - Math.max(draggedRect.left, otherRect.left);
                const overlapY = Math.min(draggedRect.bottom, otherRect.bottom) - Math.max(draggedRect.top, otherRect.top);

                if (overlapX < overlapY) {
                  if (draggedRect.left < otherRect.left) {
                    newX = otherRect.left - draggedNodeWidth;
                  } else {
                    newX = otherRect.right;
                  }
                } else {
                  if (draggedRect.top < otherRect.top) {
                    newY = otherRect.top - draggedNodeHeight;
                  } else {
                    newY = otherRect.bottom;
                  }
                }
              }
            }
          });

          return { ...node, position: { x: newX, y: newY } };
        }
        return node;
      });
      return updatedNodes;
    });
  }, [setNodes]);

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
            newEdges.push({ id: `e-${cluster.metadata.name}-${cluster.spec.controlPlaneRef.name}`, source: cluster.metadata.name, target: cluster.spec.controlPlaneRef.name, animated: true, label: 'ref' });
          }
          if (cluster.spec.workerRef) {
            newEdges.push({ id: `e-${cluster.metadata.name}-${cluster.spec.workerRef.name}`, source: cluster.metadata.name, target: cluster.spec.workerRef.name, animated: true, label: 'references worker' });
          }
        });

        talosControlPlanes.forEach((cp: any) => {
          if (cp.metadata.ownerReferences && cp.metadata.ownerReferences.length > 0) {
            const ownerRef = cp.metadata.ownerReferences.find(
              (ref: any) => ref.kind === 'TalosCluster'
            );
            if (ownerRef) {
              newEdges.push({
                id: `e-${ownerRef.name}-${cp.metadata.name}-owner`,
                source: ownerRef.name,
                target: cp.metadata.name,
                animated: true,
                label: 'owns',
              });
            }
          }
        });

        talosWorkers.forEach((worker: any) => {
          if (worker.spec.controlPlaneRef) {
            newEdges.push({ id: `e-${worker.metadata.name}-${worker.spec.controlPlaneRef.name}`, source: worker.metadata.name, target: worker.spec.controlPlaneRef.name, animated: true, label: 'ref', type: 'smoothstep' });
          }

          if (worker.metadata.ownerReferences && worker.metadata.ownerReferences.length > 0) {
            const ownerRef = worker.metadata.ownerReferences.find(
              (ref: any) => ref.kind === 'TalosCluster'
            );
            if (ownerRef) {
              newEdges.push({
                id: `e-${ownerRef.name}-${worker.metadata.name}-owner`,
                source: ownerRef.name,
                target: worker.metadata.name,
                animated: true,
                label: 'owns',
              });
            }
          }
        });

        talosMachines.forEach((machine: any) => {
          if (machine.metadata.ownerReferences && machine.metadata.ownerReferences.length > 0) {
            const owner = machine.metadata.ownerReferences[0].name;
            newEdges.push({ id: `e-${owner}-${machine.metadata.name}`, source: owner, target: machine.metadata.name, animated: true, label: 'owns' });
          }
        });

        const layoutedNodes = getLayoutedElements(newNodes, newEdges);
        setNodes(layoutedNodes);
        setEdges(newEdges);
      })
      .catch(error => {
        console.error("Error fetching resources:", error);
      });
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <Card className="talos-card talos-visualizer">
      <Card.Body>
        <div style={{ height: '600px', width: '100%', borderRadius: '8px', overflow: 'hidden' }}>
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onNodeClick={onNodeClick}
            onNodeDrag={onNodeDrag}
            fitView
          >
            <Controls />
            <Background variant={BackgroundVariant.Dots} gap={12} size={1} />
          </ReactFlow>
        </div>
        <Modal show={showModal} onHide={handleClose} className="talos-modal">
          <Modal.Header closeButton>
            <Modal.Title>{selectedNode?.metadata.name}</Modal.Title>
          </Modal.Header>
          <Modal.Body style={{ maxHeight: '400px', overflowY: 'auto' }}>
            <div className="talos-yaml-display">
              <pre>
                <code>{YAML.dump(selectedNode)}</code>
              </pre>
            </div>
          </Modal.Body>
        </Modal>
      </Card.Body>
    </Card>
  );
};

export default ClusterVisualizer;