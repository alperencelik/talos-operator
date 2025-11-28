# Config Connector Local Setup Guide

This guide explains how to install and configure Google Cloud Config Connector (KCC) on a local Kubernetes cluster (e.g., Kind, Minikube, Docker Desktop) and set up the necessary Google Service Account (GSA).

## Prerequisites

- A Google Cloud Platform (GCP) Project.
- `gcloud` CLI installed and authenticated.
- `kubectl` installed and configured for your local cluster.

## 1. Enable Config Connector API

Enable the Config Connector API in your GCP project:

```bash
gcloud services enable k8s-config-connector.googleapis.com
```

## 2. Create a Google Service Account (GSA)

Create a service account that KCC will use to manage resources:

```bash
export PROJECT_ID=$(gcloud config get-value project)
export SERVICE_ACCOUNT_NAME="kcc-local-sa"

gcloud iam service-accounts create ${SERVICE_ACCOUNT_NAME} \
    --project=${PROJECT_ID} \
    --display-name="Config Connector Local Service Account"
```

## 3. Grant Permissions

Grant the service account the `owner` role (or a more restrictive role like `editor` or specific resource admin roles) on the project:

```bash
gcloud projects add-iam-policy-binding ${PROJECT_ID} \
    --member="serviceAccount:${SERVICE_ACCOUNT_NAME}@${PROJECT_ID}.iam.gserviceaccount.com" \
    --role="roles/owner"
```

> **Note:** For production, use least-privilege roles instead of `owner`.

## 4. Create a Key for the Service Account

Since we are running locally, we cannot easily use Workload Identity. We will use a JSON key.

```bash
gcloud iam service-accounts keys create key.json \
    --iam-account=${SERVICE_ACCOUNT_NAME}@${PROJECT_ID}.iam.gserviceaccount.com
```

## 5. Install Config Connector

Download and apply the Config Connector operator manifest. We will use the "namespaced" mode which is simpler for testing.

```bash
# Download the latest operator bundle
gsutil cp gs://configconnector-operator/latest/release-bundle.tar.gz release-bundle.tar.gz
tar zxvf release-bundle.tar.gz

# Apply the operator manifests
kubectl apply -f operator-system/configconnector-operator.yaml
```

Wait for the operator pod to be ready:

```bash
kubectl wait -n configconnector-operator-system --for=condition=Ready pod --all
```

## 6. Configure Config Connector

Create a `ConfigConnector` resource to configure the operator. We will use `cluster` mode for simplicity, which uses a single Google Service Account for the entire cluster.

First, create the `cnrm-system` namespace and the secret with your key:

```bash
kubectl create namespace cnrm-system
kubectl create secret generic gcp-key \
    --from-file=key.json \
    --namespace=cnrm-system
```

Then, apply the `ConfigConnector` configuration:

```yaml
apiVersion: core.cnrm.cloud.google.com/v1beta1
kind: ConfigConnector
metadata:
  name: configconnector.core.cnrm.cloud.google.com
spec:
  mode: cluster
  credentialSecretName: gcp-key
```

Save this to `configconnector.yaml` and apply it:

```bash
kubectl apply -f configconnector.yaml
```

## 7. Verification

Verify that KCC is working by creating a simple resource, like a Pub/Sub topic (optional):

```yaml
apiVersion: pubsub.cnrm.cloud.google.com/v1beta1
kind: PubSubTopic
metadata:
  name: pubsubtopic-sample
  namespace: default
```

Check if it becomes ready:

```bash
kubectl get pubsubtopic pubsubtopic-sample -o yaml
```

You should see `status: conditions: ... type: Ready, status: "True"`.

## 8. Cleanup

To remove the key file from your local disk:

```bash
rm key.json
```
