package controller

import (
	"context"
	"fmt"
	"reflect"

	talosv1alpha1 "github.com/alperencelik/talos-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/util/jsonpath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func getMachineIPAddress(ctx context.Context, c client.Client, machine *talosv1alpha1.Machine) (*string, error) {
	logger := log.FromContext(ctx)
	if machine.Address != nil {
		return machine.Address, nil
	}
	if machine.MachineRef != nil {
		macRef := machine.MachineRef
		gvk := schema.FromAPIVersionAndKind(macRef.APIVersion, macRef.Kind)
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(gvk)

		if err := c.Get(ctx, client.ObjectKey{
			Name:      macRef.Name,
			Namespace: macRef.Namespace,
		}, obj); err != nil {
			return nil, err
		}
		// TODO: I was primarily trying with the unstructured package but it didn't worked out because can't parse without knowing the structure.
		// I should revisit this later to see if we can avoid using jsonpath for type safety but not sure how to do that yet.

		fieldPath := macRef.FieldPath
		if fieldPath == "" {
			// If no path is defined, we can't look up the IP.
			// This is logged but not treated as an error, as the IP might just not be available.
			return nil, fmt.Errorf("FieldPath is not set for machine %s", macRef.Name)
		}

		// Create a new JSONPath parser
		jp := jsonpath.New("get-ip")
		// Parse the field path, wrapping it in braces as required by the library
		if err := jp.Parse(fmt.Sprintf("{%s}", fieldPath)); err != nil {
			// The path string itself is invalid
			return nil, fmt.Errorf("invalid FieldPath for IP address %q on machine %s: %w", fieldPath, macRef.Name, err)
		}

		// Find the results from the unstructured object
		results, err := jp.FindResults(obj.Object)
		if err != nil {
			// An error occurred while executing the path query on the object
			return nil, fmt.Errorf("error executing JSONPath %q on %s %s/%s: %w",
				fieldPath, obj.GetKind(), obj.GetNamespace(), obj.GetName(), err)
		}

		// We are looking for a single string IP address.
		// results is a [][]reflect.Value. We check the first result.
		if len(results) > 0 && len(results[0]) > 0 {
			firstResult := results[0][0]

			// Ensure the result is valid and can be interfaced
			if !firstResult.IsValid() || !firstResult.CanInterface() {
				logger.Info("JSONPath query returned an invalid or un-interfaceable value",
					"machine", macRef.Name,
					"fieldPath", fieldPath)
				return nil, nil
			}

			// Get the underlying value
			value := firstResult.Interface()

			// Check if it's a string
			if ip, ok := value.(string); ok && ip != "" {
				return &ip, nil
			}
			// TODO: Control if IP is in valid format?
			// Check if it's a pointer to a string
			if ipPtr, ok := value.(*string); ok && ipPtr != nil && *ipPtr != "" {
				return ipPtr, nil
			}

			// The value was found but was not a string or was empty
			logger.Info("JSONPath query returned a non-string or empty value",
				"machine", macRef.Name,
				"fieldPath", fieldPath,
				"valueType", reflect.TypeOf(value))
		}

		// If no results were found
		logger.Info("IP address not found using JSONPath",
			"machine", macRef.Name,
			"referencedObject", obj.GetName(),
			"fieldPath", fieldPath)
		return nil, nil

		// --- End of Completed Code ---
	}
	return nil, nil
}

func getMachinesIPAddresses(ctx context.Context, c client.Client, machines *[]talosv1alpha1.Machine) ([]string, error) {
	var ips []string
	for _, machine := range *machines {
		ip, err := getMachineIPAddress(ctx, c, &machine)
		if err != nil {
			return nil, err
		}
		if ip != nil {
			ips = append(ips, *ip)
		}
	}
	return ips, nil
}
