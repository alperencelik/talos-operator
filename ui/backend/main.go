package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/yaml"
)

func main() {
	log.SetLogger(zap.New(zap.UseDevMode(true)))

	http.HandleFunc("/api/resources", getResources)
	http.HandleFunc("/api/apply", applyResource)

	fmt.Println("Server starting on port 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}

func getResources(w http.ResponseWriter, r *http.Request) {
	config, err := getKubeConfig()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting kubeconfig: %s", err), http.StatusInternalServerError)
		return
	}

	k8sClient, err := client.New(config, client.Options{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating Kubernetes client: %s", err), http.StatusInternalServerError)
		return
	}

	if err := talosv1alpha1.AddToScheme(k8sClient.Scheme()); err != nil {
		http.Error(w, fmt.Sprintf("Error adding Talos scheme: %s", err), http.StatusInternalServerError)
		return
	}

	clusters := &talosv1alpha1.TalosClusterList{}
	if err := k8sClient.List(r.Context(), clusters); err != nil {
		http.Error(w, fmt.Sprintf("Error fetching TalosClusters: %s", err), http.StatusInternalServerError)
		return
	}

	controlPlanes := &talosv1alpha1.TalosControlPlaneList{}
	if err := k8sClient.List(r.Context(), controlPlanes); err != nil {
		http.Error(w, fmt.Sprintf("Error fetching TalosControlPlanes: %s", err), http.StatusInternalServerError)
		return
	}

	workers := &talosv1alpha1.TalosWorkerList{}
	if err := k8sClient.List(r.Context(), workers); err != nil {
		http.Error(w, fmt.Sprintf("Error fetching TalosWorkers: %s", err), http.StatusInternalServerError)
		return
	}

	resources := map[string]interface{}{
		"talosClusters":      clusters.Items,
		"talosControlPlanes": controlPlanes.Items,
		"talosWorkers":       workers.Items,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resources); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding resources: %s", err), http.StatusInternalServerError)
	}
}

func applyResource(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	jsonData, err := yaml.YAMLToJSON(body)
	if err != nil {
		http.Error(w, "Error converting YAML to JSON", http.StatusBadRequest)
		return
	}

	obj := &unstructured.Unstructured{}
	if err := obj.UnmarshalJSON(jsonData); err != nil {
		http.Error(w, "Error unmarshalling JSON", http.StatusBadRequest)
		return
	}

	config, err := getKubeConfig()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting kubeconfig: %s", err), http.StatusInternalServerError)
		return
	}

	k8sClient, err := client.New(config, client.Options{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating Kubernetes client: %s", err), http.StatusInternalServerError)
		return
	}

	if err := talosv1alpha1.AddToScheme(k8sClient.Scheme()); err != nil {
		http.Error(w, fmt.Sprintf("Error adding Talos scheme: %s", err), http.StatusInternalServerError)
		return
	}

	patch := client.Apply
	force := true
	opts := &client.PatchOptions{Force: &force, FieldManager: "talos-ui"}

	if err := k8sClient.Patch(r.Context(), obj, patch, opts); err != nil {
		http.Error(w, fmt.Sprintf("Error applying resource: %s", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Resource applied successfully")
}

func getKubeConfig() (*rest.Config, error) {
	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// Fallback to kubeconfig file
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = clientcmd.RecommendedHomeFile
	}

	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}
