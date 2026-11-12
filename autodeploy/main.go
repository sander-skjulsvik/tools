// autodeploy keeps one Kubernetes deployment on the newest release of its image.
//
// It runs in the cluster next to the deployment and, every interval, lists the
// image's tags on the registry. When a vX.Y.Z tag is newer than the one the
// deployment runs, it sets that tag on the deployment. Nothing outside the
// cluster needs access to it, and its service account only needs get and patch
// on that one deployment.
//
// A rollback is a new, higher release of an older commit. To pin a version by
// hand, annotate the deployment with autodeploy/paused=true first.
//
// Configuration (environment):
//
//	AUTODEPLOY_IMAGE       image without tag, e.g. ghcr.io/owner/app (required)
//	AUTODEPLOY_DEPLOYMENT  deployment to update (required)
//	AUTODEPLOY_CONTAINER   container in it (default: the deployment's name)
//	AUTODEPLOY_NAMESPACE   (default: the pod's own namespace)
//	AUTODEPLOY_INTERVAL    time between checks (default: 5m)
//	AUTODEPLOY_AUTH_FILE   Docker config.json with registry credentials, e.g.
//	                       a mounted image pull secret (default: none)
package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// pausedAnnotation on the deployment stops autodeploy from changing it.
const pausedAnnotation = "autodeploy/paused"

type deployer struct {
	kube       *kube
	registry   *registry
	image      string // without tag
	namespace  string
	deployment string
	container  string

	reported string // last state logged at info level, to keep quiet checks quiet
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func run() error {
	image := strings.TrimSpace(os.Getenv("AUTODEPLOY_IMAGE"))
	deploymentName := os.Getenv("AUTODEPLOY_DEPLOYMENT")
	if image == "" || deploymentName == "" {
		return errors.New("AUTODEPLOY_IMAGE and AUTODEPLOY_DEPLOYMENT must be set")
	}
	host, repo, ok := strings.Cut(image, "/")
	if !ok || !strings.Contains(host, ".") || strings.ContainsAny(repo, ":@") {
		return fmt.Errorf("AUTODEPLOY_IMAGE must look like registry.example.com/owner/app, without a tag: %q", image)
	}
	interval := 5 * time.Minute
	if s := os.Getenv("AUTODEPLOY_INTERVAL"); s != "" {
		d, err := time.ParseDuration(s)
		if err != nil || d < 10*time.Second {
			return fmt.Errorf("AUTODEPLOY_INTERVAL must be a duration of at least 10s: %q", s)
		}
		interval = d
	}
	k, err := inCluster()
	if err != nil {
		return err
	}
	d := &deployer{
		kube: k,
		registry: &registry{
			client:   &http.Client{Timeout: 30 * time.Second},
			base:     "https://" + host,
			host:     host,
			repo:     repo,
			authFile: os.Getenv("AUTODEPLOY_AUTH_FILE"),
		},
		image:      image,
		namespace:  cmp.Or(os.Getenv("AUTODEPLOY_NAMESPACE"), inClusterNamespace()),
		deployment: deploymentName,
		container:  cmp.Or(os.Getenv("AUTODEPLOY_CONTAINER"), deploymentName),
	}
	if d.namespace == "" {
		return errors.New("AUTODEPLOY_NAMESPACE is not set and the pod's namespace is unknown")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	slog.Info("starting", "image", image, "deployment", d.namespace+"/"+d.deployment, "container", d.container, "interval", interval.String())
	for {
		if err := d.check(ctx); err != nil && ctx.Err() == nil {
			slog.Error("check failed", "error", err)
			d.reported = ""
		}
		select {
		case <-ctx.Done():
			slog.Info("stopping")
			return nil
		case <-time.After(interval):
		}
	}
}

// check rolls the deployment forward if the registry has a newer release.
func (d *deployer) check(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	dep, err := d.kube.getDeployment(ctx, d.namespace, d.deployment)
	if err != nil {
		return fmt.Errorf("reading the deployment: %w", err)
	}
	if dep.Metadata.Annotations[pausedAnnotation] == "true" {
		d.report("paused", "paused, not changing the deployment", "annotation", pausedAnnotation)
		return nil
	}
	var current string
	found := false
	for _, c := range dep.Spec.Template.Spec.Containers {
		if c.Name == d.container {
			current, found = c.Image, true
		}
	}
	if !found {
		return fmt.Errorf("the deployment has no container %q", d.container)
	}
	currentTag, ok := strings.CutPrefix(current, d.image+":")
	if !ok {
		return fmt.Errorf("container %q runs %q, which isn't a tag of %s; not touching it", d.container, current, d.image)
	}

	tags, err := d.registry.tags(ctx)
	if err != nil {
		return err
	}
	latest, ok := newest(tags)
	if !ok {
		return fmt.Errorf("no vX.Y.Z tags on %s", d.image)
	}
	if running, ok := parseVersion(currentTag); ok && !running.less(latest) {
		d.report("up to date "+currentTag+" "+latest.String(), "up to date", "running", currentTag, "newest", latest.String())
		return nil
	}

	target := d.image + ":" + latest.String()
	if err := d.kube.setImage(ctx, d.namespace, d.deployment, d.container, target, "autodeploy: "+latest.String()); err != nil {
		return fmt.Errorf("setting %s: %w", target, err)
	}
	slog.Info("rolled out", "from", currentTag, "to", latest.String())
	d.reported = ""
	return nil
}

// report logs at info level when the state changes, and at debug otherwise.
func (d *deployer) report(state, msg string, args ...any) {
	if state == d.reported {
		slog.Debug(msg, args...)
		return
	}
	d.reported = state
	slog.Info(msg, args...)
}
