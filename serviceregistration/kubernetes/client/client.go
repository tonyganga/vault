package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/hashicorp/go-cleanhttp"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrNotInCluster = errors.New("unable to load in-cluster configuration, KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT must be defined")
)

type Pod struct {
	Metadata map[string]interface{}
}

type Patch struct {
	Path, Value string
}

func New() (*Client, error) {
	config, err := inClusterConfig()
	if err != nil {
		return nil, err
	}
	return &Client{
		config: config,
	}, nil
}

type Client struct {
	config *Config
}

// GetPod merely verifies a pod's existence, returning an
// error if the pod doesn't exist.
func (c *Client) GetPod(namespace, podName string) (*Pod, error) {
	endpoint := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s", namespace, podName)
	method := http.MethodGet

	req, err := http.NewRequest(method, c.config.Host+endpoint, nil)
	if err != nil {
		return nil, err
	}
	pod := &Pod{}
	if err := c.do(req, pod); err != nil {
		return nil, err
	}
	return pod, nil
}

// PatchPod updates the pod's tags to the given ones,
// overwriting previous values for a given tag key. It does so
// non-destructively, or in other words, without tearing down
// the pod.
func (c *Client) PatchPod(namespace, podName string, patches ...*Patch) error {
	endpoint := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s", namespace, podName)
	method := http.MethodPatch

	var jsonPatches []interface{}
	for _, patch := range patches {
		jsonPatches = append(jsonPatches, map[string]string{
			"op":    "add",
			"path":  patch.Path,
			"value": patch.Value,
		})
	}
	body, err := json.Marshal(jsonPatches)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(method, c.config.Host+endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json-patch+json")
	return c.do(req, nil)
}

func (c *Client) do(req *http.Request, ptrToReturnObj interface{}) error {
	// Finish setting up a valid request.
	req.Header.Set("Authorization", "Bearer "+c.config.BearerToken)
	req.Header.Set("Accept", "application/json")
	client := cleanhttp.DefaultClient()
	client.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: c.config.CACertPool,
		},
	}

	haveTriedOrigToken := false
	haveTriedNewToken := false
	for !haveTriedOrigToken || !haveTriedNewToken {
		// Execute it.
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		if !haveTriedOrigToken {
			haveTriedOrigToken = true
		} else {
			haveTriedNewToken = true
		}

		// Check for success.
		switch resp.StatusCode {
		case 200, 201, 202:
			// Pass.
			break
		case 401, 403:
			// Perhaps the token from our bearer token file has been refreshed.
			config, err := inClusterConfig()
			if err != nil {
				return err
			}
			c.config = config
			// Continue to try again.
			continue
		case 404:
			return ErrNotFound
		default:
			return fmt.Errorf("unexpected status code: %s", sanitizedDebuggingInfo(req, resp))
		}

		// If we're not supposed to read out the body, we have nothing further
		// to do here.
		if ptrToReturnObj == nil {
			return nil
		}

		// Attempt to read out the body into the given return object.
		if err := json.NewDecoder(resp.Body).Decode(ptrToReturnObj); err != nil {
			return fmt.Errorf("unable to read as %T: %s", ptrToReturnObj, sanitizedDebuggingInfo(req, resp))
		}
	}
	return nil
}

// sanitizedDebuggingInfo converts an http response to a string without
// including its headers, to avoid leaking authorization
// headers.
func sanitizedDebuggingInfo(req *http.Request, resp *http.Response) string {
	// Ignore error here because if we're unable to read the body or
	// it doesn't exist, it'll just be "", which is fine.
	body, _ := ioutil.ReadAll(resp.Body)
	return fmt.Sprintf("method: %s, url: %s, statuscode: %d, body: %s", req.Method, req.URL, resp.StatusCode, body)
}
