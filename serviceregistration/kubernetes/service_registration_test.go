package kubernetes

import (
	"testing"

	"github.com/hashicorp/go-hclog"
	sr "github.com/hashicorp/vault/serviceregistration"
	"github.com/hashicorp/vault/serviceregistration/kubernetes/client"
)

var testVersion = "version 1"

func TestServiceRegistration(t *testing.T) {
	currentLabels, closeFunc := client.TestServer(t)
	defer closeFunc()

	if len(currentLabels) != 0 {
		t.Fatalf("expected 0 current labels but have %d: %s", len(currentLabels), currentLabels)
	}
	shutdownCh := make(chan struct{})
	config := map[string]string{
		"namespace": client.TestNamespace,
		"pod_name":  client.TestPodname,
	}
	logger := hclog.NewNullLogger()
	state := &sr.State{
		VaultVersion:         testVersion,
		IsInitialized:        true,
		IsSealed:             true,
		IsActive:             true,
		IsPerformanceStandby: true,
	}
	reg, err := NewServiceRegistration(shutdownCh, config, logger, state, "")
	if err != nil {
		t.Fatal(err)
	}

	// Test initial state.
	if len(currentLabels) != 5 {
		t.Fatalf("expected 5 current labels but have %d: %s", len(currentLabels), currentLabels)
	}
	if currentLabels[pathToLabels+labelVaultVersion] != testVersion {
		t.Fatalf("expected %q but received %q", testVersion, currentLabels[pathToLabels+labelVaultVersion])
	}
	if currentLabels[pathToLabels+labelActive] != toString(true) {
		t.Fatalf("expected %q but received %q", toString(true), currentLabels[pathToLabels+labelVaultVersion])
	}
	if currentLabels[pathToLabels+labelSealed] != toString(true) {
		t.Fatalf("expected %q but received %q", toString(true), currentLabels[pathToLabels+labelVaultVersion])
	}
	if currentLabels[pathToLabels+labelPerfStandby] != toString(true) {
		t.Fatalf("expected %q but received %q", toString(true), currentLabels[pathToLabels+labelVaultVersion])
	}
	if currentLabels[pathToLabels+labelInitialized] != toString(true) {
		t.Fatalf("expected %q but received %q", toString(true), currentLabels[pathToLabels+labelVaultVersion])
	}

	// Test NotifyActiveStateChange.
	if err := reg.NotifyActiveStateChange(false); err != nil {
		t.Fatal(err)
	}
	if currentLabels[pathToLabels+labelActive] != toString(false) {
		t.Fatalf("expected %q but received %q", toString(false), currentLabels[pathToLabels+labelActive])
	}
	if err := reg.NotifyActiveStateChange(true); err != nil {
		t.Fatal(err)
	}
	if currentLabels[pathToLabels+labelActive] != toString(true) {
		t.Fatalf("expected %q but received %q", toString(true), currentLabels[pathToLabels+labelActive])
	}

	// Test NotifySealedStateChange.
	if err := reg.NotifySealedStateChange(false); err != nil {
		t.Fatal(err)
	}
	if currentLabels[pathToLabels+labelSealed] != toString(false) {
		t.Fatalf("expected %q but received %q", toString(false), currentLabels[pathToLabels+labelSealed])
	}
	if err := reg.NotifySealedStateChange(true); err != nil {
		t.Fatal(err)
	}
	if currentLabels[pathToLabels+labelSealed] != toString(true) {
		t.Fatalf("expected %q but received %q", toString(true), currentLabels[pathToLabels+labelSealed])
	}

	// Test NotifyPerformanceStandbyStateChange.
	if err := reg.NotifyPerformanceStandbyStateChange(false); err != nil {
		t.Fatal(err)
	}
	if currentLabels[pathToLabels+labelPerfStandby] != toString(false) {
		t.Fatalf("expected %q but received %q", toString(false), currentLabels[pathToLabels+labelPerfStandby])
	}
	if err := reg.NotifyPerformanceStandbyStateChange(true); err != nil {
		t.Fatal(err)
	}
	if currentLabels[pathToLabels+labelPerfStandby] != toString(true) {
		t.Fatalf("expected %q but received %q", toString(true), currentLabels[pathToLabels+labelPerfStandby])
	}

	// Test NotifyInitializedStateChange.
	if err := reg.NotifyInitializedStateChange(false); err != nil {
		t.Fatal(err)
	}
	if currentLabels[pathToLabels+labelInitialized] != toString(false) {
		t.Fatalf("expected %q but received %q", toString(false), currentLabels[pathToLabels+labelInitialized])
	}
	if err := reg.NotifyInitializedStateChange(true); err != nil {
		t.Fatal(err)
	}
	if currentLabels[pathToLabels+labelInitialized] != toString(true) {
		t.Fatalf("expected %q but received %q", toString(true), currentLabels[pathToLabels+labelInitialized])
	}
}
