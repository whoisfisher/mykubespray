package kubernetes

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"time"
)

func getNamespace(obj map[string]interface{}) string {
	if metadata, ok := obj["metadata"].(map[string]interface{}); ok {
		if namespace, ok := metadata["namespace"].(string); ok {
			return namespace
		}
	}
	return ""
}

func applyNonNamespaced(resourceClient dynamic.ResourceInterface, obj map[string]interface{}) error {
	name := obj["metadata"].(map[string]interface{})["name"].(string)
	if name == "" {
		return fmt.Errorf("name not found in YAML")
	}

	return applyResourceWithRetry(resourceClient, name, obj)
}

func applyInNamespace(resourceClient dynamic.NamespaceableResourceInterface, namespace string, obj map[string]interface{}) error {
	name := obj["metadata"].(map[string]interface{})["name"].(string)
	if name == "" {
		return fmt.Errorf("name not found in YAML")
	}

	return applyNamespacedResourceWithRetry(resourceClient, name, obj)
}

func createOrUpdateNamespacedResource(resourceClient dynamic.NamespaceableResourceInterface, name string, obj map[string]interface{}) error {
	_, err := resourceClient.Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = resourceClient.Create(context.TODO(), &unstructured.Unstructured{Object: obj}, v1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("error creating resource: %w", err)
			}
			return nil
		}
		return fmt.Errorf("error checking resource existence: %w", err)
	}

	_, err = resourceClient.Update(context.TODO(), &unstructured.Unstructured{Object: obj}, v1.UpdateOptions{})
	if err != nil {
		if errors.IsConflict(err) {
			return fmt.Errorf("resource update conflict: %w", err)
		}
		return fmt.Errorf("error updating resource: %w", err)
	}
	return nil
}

func createOrUpdateResource(resourceClient dynamic.ResourceInterface, name string, obj map[string]interface{}) error {
	_, err := resourceClient.Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = resourceClient.Create(context.TODO(), &unstructured.Unstructured{Object: obj}, v1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("error creating resource: %w", err)
			}
			return nil
		}
		return fmt.Errorf("error checking resource existence: %w", err)
	}

	_, err = resourceClient.Update(context.TODO(), &unstructured.Unstructured{Object: obj}, v1.UpdateOptions{})
	if err != nil {
		if errors.IsConflict(err) {
			return fmt.Errorf("resource update conflict: %w", err)
		}
		return fmt.Errorf("error updating resource: %w", err)
	}
	return nil
}

func applyNamespacedResourceWithRetry(resourceClient dynamic.NamespaceableResourceInterface, name string, obj map[string]interface{}) error {
	return wait.ExponentialBackoff(wait.Backoff{Steps: 5, Duration: time.Second, Factor: 2}, func() (bool, error) {
		err := createOrUpdateNamespacedResource(resourceClient, name, obj)
		if err == nil {
			return true, nil
		}
		if isTemporaryError(err) {
			return false, nil
		}
		return true, err
	})
}

func applyResourceWithRetry(resourceClient dynamic.ResourceInterface, name string, obj map[string]interface{}) error {
	return wait.ExponentialBackoff(wait.Backoff{Steps: 5, Duration: time.Second, Factor: 2}, func() (bool, error) {
		err := createOrUpdateResource(resourceClient, name, obj)
		if err == nil {
			return true, nil
		}
		if isTemporaryError(err) {
			return false, nil
		}
		return true, err
	})
}

func isTemporaryError(err error) bool {
	return errors.IsServerTimeout(err) || errors.IsTimeout(err)
}
