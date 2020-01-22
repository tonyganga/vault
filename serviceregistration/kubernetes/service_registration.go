package kubernetes

import (
	"fmt"
	"os"

	log "github.com/hashicorp/go-hclog"
	sr "github.com/hashicorp/vault/serviceregistration"
	"github.com/hashicorp/vault/serviceregistration/kubernetes/client"
)

const (
	// Labels are placed in a pod's metadata.
	labelVaultVersion = "vault-version"
	labelActive       = "vault-ha-active"
	labelSealed       = "vault-ha-sealed"
	labelPerfStandby  = "vault-ha-perf-standby"
	labelInitialized  = "vault-ha-initialized"

	// This is the path to where these labels are applied.
	pathToLabels = "/metadata/labels/"
)

func NewServiceRegistration(shutdownCh <-chan struct{}, config map[string]string, logger log.Logger, state *sr.State, _ string) (sr.ServiceRegistration, error) {
	c, err := client.New()
	if err != nil {
		return nil, err
	}

	namespace := ""
	switch {
	case os.Getenv(client.EnvVarKubernetesNamespace) != "":
		namespace = os.Getenv(client.EnvVarKubernetesNamespace)
	case config["namespace"] != "":
		namespace = config["namespace"]
	default:
		return nil, fmt.Errorf(`namespace must be provided via %q or the "namespace" config parameter`, client.EnvVarKubernetesNamespace)
	}
	if logger.IsDebug() {
		logger.Debug(fmt.Sprintf("namespace: %q", namespace))
	}

	podName := ""
	switch {
	case os.Getenv(client.EnvVarKubernetesPodName) != "":
		podName = os.Getenv(client.EnvVarKubernetesPodName)
	case config["pod_name"] != "":
		podName = config["pod_name"]
	default:
		return nil, fmt.Errorf(`pod name must be provided via %q or the "pod_name" config parameter`, client.EnvVarKubernetesPodName)
	}
	if logger.IsDebug() {
		logger.Debug(fmt.Sprintf("pod name: %q", podName))
	}

	// Verify that the pod exists and our configuration looks good.
	pod, err := c.GetPod(namespace, podName)
	if err != nil {
		return nil, err
	}
	if pod.Metadata["labels"] == nil {
		return nil, fmt.Errorf(`pod name %q in namespace %q must have "/metadata/labels" to be usable for service registration`, podName, namespace)
	}

	// Perform an initial labelling of Vault as it starts up.
	patches := []*client.Patch{
		{Path: pathToLabels + labelVaultVersion, Value: state.VaultVersion},
		{Path: pathToLabels + labelActive, Value: toString(state.IsActive)},
		{Path: pathToLabels + labelSealed, Value: toString(state.IsSealed)},
		{Path: pathToLabels + labelPerfStandby, Value: toString(state.IsPerformanceStandby)},
		{Path: pathToLabels + labelInitialized, Value: toString(state.IsInitialized)},
	}
	if err := c.PatchPod(namespace, podName, patches...); err != nil {
		return nil, err
	}
	registration := &serviceRegistration{
		logger:    logger,
		podName:   podName,
		namespace: namespace,
		client:    c,
	}

	// Run a background goroutine to leave labels in the final state we'd like
	// when Vault shuts down.
	go registration.onShutdown(shutdownCh)
	return registration, nil
}

type serviceRegistration struct {
	logger             log.Logger
	namespace, podName string
	client             *client.Client
}

func (r *serviceRegistration) NotifyActiveStateChange(isActive bool) error {
	return r.client.PatchPod(r.namespace, r.podName, &client.Patch{
		Path:  pathToLabels + labelActive,
		Value: toString(isActive),
	})
}

func (r *serviceRegistration) NotifySealedStateChange(isSealed bool) error {
	return r.client.PatchPod(r.namespace, r.podName, &client.Patch{
		Path:  pathToLabels + labelSealed,
		Value: toString(isSealed),
	})
}

func (r *serviceRegistration) NotifyPerformanceStandbyStateChange(isStandby bool) error {
	return r.client.PatchPod(r.namespace, r.podName, &client.Patch{
		Path:  pathToLabels + labelPerfStandby,
		Value: toString(isStandby),
	})
}

func (r *serviceRegistration) NotifyInitializedStateChange(isInitialized bool) error {
	return r.client.PatchPod(r.namespace, r.podName, &client.Patch{
		Path:  pathToLabels + labelInitialized,
		Value: toString(isInitialized),
	})
}

func (r *serviceRegistration) onShutdown(shutdownCh <-chan struct{}) {
	<-shutdownCh

	// Label the pod with the values we want to leave behind after shutdown.
	patches := []*client.Patch{
		{Path: pathToLabels + labelActive, Value: toString(false)},
		{Path: pathToLabels + labelSealed, Value: toString(true)},
		{Path: pathToLabels + labelPerfStandby, Value: toString(false)},
		{Path: pathToLabels + labelInitialized, Value: toString(false)},
	}
	if err := r.client.PatchPod(r.namespace, r.podName, patches...); err != nil {
		if r.logger.IsError() {
			r.logger.Error(fmt.Sprintf("unable to set final status on pod name %q in namespace %q on shutdown: %s", r.podName, r.namespace, err))
		}
		return
	}
}

// Converts a bool to "true" or "false".
func toString(b bool) string {
	return fmt.Sprintf("%t", b)
}
