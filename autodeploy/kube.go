package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

const serviceAccountDir = "/var/run/secrets/kubernetes.io/serviceaccount"

// kube talks to the Kubernetes API with the pod's service account. It only
// needs get and patch on one deployment.
type kube struct {
	client    *http.Client
	base      string // e.g. https://10.43.0.1:443
	tokenFile string // read on every request: projected tokens are rotated
}

func inCluster() (*kube, error) {
	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	if host == "" || port == "" {
		return nil, errors.New("not running in a cluster (KUBERNETES_SERVICE_HOST is not set)")
	}
	ca, err := os.ReadFile(serviceAccountDir + "/ca.crt")
	if err != nil {
		return nil, fmt.Errorf("reading the cluster CA: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(ca) {
		return nil, errors.New("no certificates in the cluster CA file")
	}
	return &kube{
		client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}},
		},
		base:      "https://" + net.JoinHostPort(host, port),
		tokenFile: serviceAccountDir + "/token",
	}, nil
}

// inClusterNamespace is the namespace the pod runs in.
func inClusterNamespace() string {
	ns, err := os.ReadFile(serviceAccountDir + "/namespace")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(ns))
}

type container struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}

type deployment struct {
	Metadata struct {
		Annotations map[string]string `json:"annotations"`
	} `json:"metadata"`
	Spec struct {
		Template struct {
			Spec struct {
				Containers []container `json:"containers"`
			} `json:"spec"`
		} `json:"template"`
	} `json:"spec"`
}

func (k *kube) path(namespace, name string) string {
	return k.base + "/apis/apps/v1/namespaces/" + namespace + "/deployments/" + name
}

func (k *kube) do(ctx context.Context, method, url, contentType string, body []byte) (*http.Response, error) {
	token, err := os.ReadFile(k.tokenFile)
	if err != nil {
		return nil, fmt.Errorf("reading the service account token: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(string(token)))
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return k.client.Do(req)
}

func (k *kube) getDeployment(ctx context.Context, namespace, name string) (deployment, error) {
	var d deployment
	resp, err := k.do(ctx, http.MethodGet, k.path(namespace, name), "", nil)
	if err != nil {
		return d, err
	}
	return d, decode(resp, &d)
}

// setImage changes one container's image. The change-cause annotation shows
// up in `kubectl rollout history`.
func (k *kube) setImage(ctx context.Context, namespace, name, containerName, image, cause string) error {
	patch := map[string]any{
		"metadata": map[string]any{
			"annotations": map[string]string{"kubernetes.io/change-cause": cause},
		},
		"spec": map[string]any{
			"template": map[string]any{
				"spec": map[string]any{
					"containers": []container{{Name: containerName, Image: image}},
				},
			},
		},
	}
	body, err := json.Marshal(patch)
	if err != nil {
		return err
	}
	resp, err := k.do(ctx, http.MethodPatch, k.path(namespace, name), "application/strategic-merge-patch+json", body)
	if err != nil {
		return err
	}
	var ignored json.RawMessage
	return decode(resp, &ignored)
}
