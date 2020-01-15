package client

import (
	"testing"
)

func TestClient(t *testing.T) {
	currentLabels, closeFunc := TestServer(t)
	defer closeFunc()

	client, err := New()
	if err != nil {
		t.Fatal(err)
	}
	e := &env{
		client:         client,
		currentPatches: currentLabels,
	}
	e.TestGetPod(t)
	e.TestGetPodNotFound(t)
	e.TestUpdatePodTags(t)
	e.TestUpdatePodTagsNotFound(t)
}

type env struct {
	client         *Client
	currentPatches map[string]string
}

func (e *env) TestGetPod(t *testing.T) {
	pod, err := e.client.GetPod(TestNamespace, TestPodname)
	if err != nil {
		t.Fatal(err)
	}
	if pod.Metadata["name"] != "shell-demo" {
		t.Fatalf("expected %q but received %q", "shell-demo", pod.Metadata["name"])
	}
}

func (e *env) TestGetPodNotFound(t *testing.T) {
	_, err := e.client.GetPod(TestNamespace, "no-exist")
	if err == nil {
		t.Fatal("expected error because pod is unfound")
	}
	if err != ErrNotFound {
		t.Fatalf("expected %q but received %q", ErrNotFound, err)
	}
}

func (e *env) TestUpdatePodTags(t *testing.T) {
	if err := e.client.PatchPod(TestNamespace, TestPodname, &Patch{
		Path:  "/metadata/labels/fizz",
		Value: "buzz",
	}); err != nil {
		t.Fatal(err)
	}
	if len(e.currentPatches) != 1 {
		t.Fatalf("expected 1 label but received %q", e.currentPatches)
	}
	if e.currentPatches["/metadata/labels/fizz"] != "buzz" {
		t.Fatalf("expected buzz but received %q", e.currentPatches["fizz"])
	}
}

func (e *env) TestUpdatePodTagsNotFound(t *testing.T) {
	err := e.client.PatchPod(TestNamespace, "no-exist", &Patch{
		Path:  "/metadata/labels/fizz",
		Value: "buzz",
	})
	if err == nil {
		t.Fatal("expected error because pod is unfound")
	}
	if err != ErrNotFound {
		t.Fatalf("expected %q but received %q", ErrNotFound, err)
	}
}
