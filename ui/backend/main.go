package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/yaml"
)

const talosAPIVersion = "talos.alperen.cloud/v1alpha1"

// allowedKinds gates writes and deletes to the Talos CRDs the UI knows about.
// Anything else is rejected so the UI cannot be used to delete unrelated cluster
// resources via a crafted request.
var allowedKinds = map[string]struct{}{
	"TalosCluster":             {},
	"TalosControlPlane":        {},
	"TalosWorker":              {},
	"TalosMachine":             {},
	"TalosClusterAddon":        {},
	"TalosClusterAddonRelease": {},
	"TalosEtcdBackup":          {},
	"TalosEtcdBackupSchedule":  {},
}

func removeMetadata(meta *metav1.ObjectMeta) {
	meta.UID = ""
	meta.ResourceVersion = ""
	meta.ManagedFields = nil
	if meta.Annotations != nil {
		delete(meta.Annotations, "kubectl.kubernetes.io/last-applied-configuration")
	}
}

func main() {
	log.SetLogger(zap.New(zap.UseDevMode(true)))

	http.HandleFunc("/api/resources", getResources)
	http.HandleFunc("/api/resources/", resourceByPath)
	http.HandleFunc("/api/events/", eventsByPath)
	http.HandleFunc("/api/apply", applyResource)

	http.HandleFunc("/", serveStaticOrIndex)

	port := os.Getenv("UI_PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Starting server on port %s...\n", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}

// newClient builds a controller-runtime client with the Talos scheme registered.
func newClient() (client.Client, error) {
	config, err := getKubeConfig()
	if err != nil {
		return nil, fmt.Errorf("getting kubeconfig: %w", err)
	}
	c, err := client.New(config, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("creating client: %w", err)
	}
	if err := talosv1alpha1.AddToScheme(c.Scheme()); err != nil {
		return nil, fmt.Errorf("registering scheme: %w", err)
	}
	return c, nil
}

func getResources(w http.ResponseWriter, r *http.Request) {
	k8sClient, err := newClient()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

	machines := &talosv1alpha1.TalosMachineList{}
	if err := k8sClient.List(r.Context(), machines); err != nil {
		http.Error(w, fmt.Sprintf("Error fetching TalosMachines: %s", err), http.StatusInternalServerError)
		return
	}

	addons := &talosv1alpha1.TalosClusterAddonList{}
	if err := k8sClient.List(r.Context(), addons); err != nil {
		http.Error(w, fmt.Sprintf("Error fetching TalosClusterAddons: %s", err), http.StatusInternalServerError)
		return
	}

	addonReleases := &talosv1alpha1.TalosClusterAddonReleaseList{}
	if err := k8sClient.List(r.Context(), addonReleases); err != nil {
		http.Error(w, fmt.Sprintf("Error fetching TalosClusterAddonReleases: %s", err), http.StatusInternalServerError)
		return
	}

	etcdBackups := &talosv1alpha1.TalosEtcdBackupList{}
	if err := k8sClient.List(r.Context(), etcdBackups); err != nil {
		http.Error(w, fmt.Sprintf("Error fetching TalosEtcdBackups: %s", err), http.StatusInternalServerError)
		return
	}

	etcdBackupSchedules := &talosv1alpha1.TalosEtcdBackupScheduleList{}
	if err := k8sClient.List(r.Context(), etcdBackupSchedules); err != nil {
		http.Error(w, fmt.Sprintf("Error fetching TalosEtcdBackupSchedules: %s", err), http.StatusInternalServerError)
		return
	}

	for i := range clusters.Items {
		removeMetadata(&clusters.Items[i].ObjectMeta)
	}
	for i := range controlPlanes.Items {
		removeMetadata(&controlPlanes.Items[i].ObjectMeta)
	}
	for i := range workers.Items {
		removeMetadata(&workers.Items[i].ObjectMeta)
	}
	for i := range machines.Items {
		removeMetadata(&machines.Items[i].ObjectMeta)
	}
	for i := range addons.Items {
		removeMetadata(&addons.Items[i].ObjectMeta)
	}
	for i := range addonReleases.Items {
		removeMetadata(&addonReleases.Items[i].ObjectMeta)
	}
	for i := range etcdBackups.Items {
		removeMetadata(&etcdBackups.Items[i].ObjectMeta)
	}
	for i := range etcdBackupSchedules.Items {
		removeMetadata(&etcdBackupSchedules.Items[i].ObjectMeta)
	}

	resources := map[string]any{
		"talosClusters":             clusters.Items,
		"talosControlPlanes":        controlPlanes.Items,
		"talosWorkers":              workers.Items,
		"talosMachines":             machines.Items,
		"talosClusterAddons":        addons.Items,
		"talosClusterAddonReleases": addonReleases.Items,
		"talosEtcdBackups":          etcdBackups.Items,
		"talosEtcdBackupSchedules":  etcdBackupSchedules.Items,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resources); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding resources: %s", err), http.StatusInternalServerError)
	}
}

// applyResource accepts a YAML manifest in the request body and uses server-side
// apply so it works for both create and update without needing resourceVersion.
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

	if _, ok := allowedKinds[obj.GetKind()]; !ok {
		http.Error(w, fmt.Sprintf("Kind %q is not managed by this UI", obj.GetKind()), http.StatusBadRequest)
		return
	}

	// Clear server-managed fields so we can submit the same object we received
	// from /api/resources (which strips them) or one freshly authored in the UI.
	obj.SetResourceVersion("")
	obj.SetManagedFields(nil)

	k8sClient, err := newClient()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(obj.GroupVersionKind())
	getErr := k8sClient.Get(r.Context(), client.ObjectKeyFromObject(obj), existing)
	switch {
	case apierrors.IsNotFound(getErr):
		if err := k8sClient.Create(r.Context(), obj); err != nil {
			http.Error(w, fmt.Sprintf("Error creating resource: %s", err), http.StatusInternalServerError)
			return
		}
	case getErr != nil:
		http.Error(w, fmt.Sprintf("Error reading existing resource: %s", getErr), http.StatusInternalServerError)
		return
	default:
		obj.SetResourceVersion(existing.GetResourceVersion())
		if err := k8sClient.Update(r.Context(), obj); err != nil {
			http.Error(w, fmt.Sprintf("Error updating resource: %s", err), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Resource applied successfully")
}

// parseKindNamespaceName extracts {kind}/{namespace}/{name} from a URL path that
// starts with prefix. Returns "", "", "", false if the path doesn't match.
func parseKindNamespaceName(urlPath, prefix string) (kind, namespace, name string, ok bool) {
	rest := strings.TrimPrefix(urlPath, prefix)
	parts := strings.Split(rest, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}

// resourceByPath routes GET and DELETE for /api/resources/{kind}/{namespace}/{name}.
func resourceByPath(w http.ResponseWriter, r *http.Request) {
	kind, namespace, name, ok := parseKindNamespaceName(r.URL.Path, "/api/resources/")
	if !ok {
		http.Error(w, "expected /api/resources/{kind}/{namespace}/{name}", http.StatusBadRequest)
		return
	}
	if _, ok := allowedKinds[kind]; !ok {
		http.Error(w, fmt.Sprintf("Kind %q is not managed by this UI", kind), http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		getResource(w, r, kind, namespace, name)
	case http.MethodDelete:
		deleteResource(w, r, kind, namespace, name)
	default:
		w.Header().Set("Allow", "GET, DELETE")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func getResource(w http.ResponseWriter, r *http.Request, kind, namespace, name string) {
	k8sClient, err := newClient()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(talosAPIVersion)
	obj.SetKind(kind)
	if err := k8sClient.Get(r.Context(), client.ObjectKey{Namespace: namespace, Name: name}, obj); err != nil {
		if apierrors.IsNotFound(err) {
			http.Error(w, "resource not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Error fetching resource: %s", err), http.StatusInternalServerError)
		return
	}

	// Strip server-managed fields for a cleaner UI payload.
	obj.SetManagedFields(nil)
	annotations := obj.GetAnnotations()
	if annotations != nil {
		delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")
		obj.SetAnnotations(annotations)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(obj.Object); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding resource: %s", err), http.StatusInternalServerError)
	}
}

func deleteResource(w http.ResponseWriter, r *http.Request, kind, namespace, name string) {
	k8sClient, err := newClient()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(talosAPIVersion)
	obj.SetKind(kind)
	obj.SetNamespace(namespace)
	obj.SetName(name)

	if err := k8sClient.Delete(r.Context(), obj); err != nil {
		if apierrors.IsNotFound(err) {
			http.Error(w, "resource not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Error deleting resource: %s", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// eventsByPath handles GET /api/events/{kind}/{namespace}/{name} and returns the
// core/v1 events whose involvedObject matches the given kind+name in the namespace.
func eventsByPath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	kind, namespace, name, ok := parseKindNamespaceName(r.URL.Path, "/api/events/")
	if !ok {
		http.Error(w, "expected /api/events/{kind}/{namespace}/{name}", http.StatusBadRequest)
		return
	}
	if _, ok := allowedKinds[kind]; !ok {
		http.Error(w, fmt.Sprintf("Kind %q is not managed by this UI", kind), http.StatusBadRequest)
		return
	}

	k8sClient, err := newClient()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	events := &corev1.EventList{}
	if err := k8sClient.List(r.Context(), events, client.InNamespace(namespace)); err != nil {
		http.Error(w, fmt.Sprintf("Error fetching events: %s", err), http.StatusInternalServerError)
		return
	}

	filtered := make([]corev1.Event, 0, len(events.Items))
	for _, e := range events.Items {
		if e.InvolvedObject.Kind == kind && e.InvolvedObject.Name == name {
			removeMetadata(&e.ObjectMeta)
			filtered = append(filtered, e)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(filtered); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding events: %s", err), http.StatusInternalServerError)
	}
}

// serveStaticOrIndex serves files from ./web; if the requested path isn't a file,
// it falls back to ./web/index.html so client-side routes deep-link correctly.
// Any /api/* path that reaches this handler is treated as 404 — the API routes
// have their own handlers and should not fall through to the SPA shell.
func serveStaticOrIndex(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}

	cleaned := path.Clean(r.URL.Path)
	if cleaned == "/" {
		http.ServeFile(w, r, "./web/index.html")
		return
	}

	diskPath := filepath.Join("./web", cleaned)
	if info, err := os.Stat(diskPath); err == nil && !info.IsDir() {
		http.ServeFile(w, r, diskPath)
		return
	}

	http.ServeFile(w, r, "./web/index.html")
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
