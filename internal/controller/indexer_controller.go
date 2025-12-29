/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	torrentsv1alpha1 "vitoru.fun/torrents/api/v1alpha1"
)

// IndexerReconciler reconciles a Indexer object
type IndexerReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	HTTPClient      *http.Client
	FlareSolverrURL string
}

// +kubebuilder:rbac:groups=torrents.vitoru.fun,resources=indexers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=torrents.vitoru.fun,resources=indexers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=torrents.vitoru.fun,resources=indexers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *IndexerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	// Fetch the Indexer instance
	var indexer torrentsv1alpha1.Indexer
	if err := r.Get(ctx, req.NamespacedName, &indexer); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Initialize status if needed
	if indexer.Status.Conditions == nil {
		indexer.Status.Conditions = []metav1.Condition{}
	}

	l.Info("Reconciling Indexer", "name", indexer.Name)

	needsCheck := true
	// Check if we need to run health check (e.g., every 15 minutes)
	readyCondition := apimeta.FindStatusCondition(indexer.Status.Conditions, "Ready")
	if readyCondition != nil {
		if time.Since(readyCondition.LastTransitionTime.Time) < 15*time.Minute {
			needsCheck = false
		}
	}

	if needsCheck {
		l.Info("Running health check", "indexer", indexer.Name)
		healthy, errMsg := r.checkHealth(ctx, &indexer)

		status := metav1.ConditionTrue
		reason := "HealthCheckSucceeded"
		message := "Indexer is healthy"

		if !healthy {
			status = metav1.ConditionFalse
			reason = "HealthCheckFailed"
			message = errMsg
		}

		newCondition := metav1.Condition{
			Type:               "Ready",
			Status:             status,
			Reason:             reason,
			Message:            message,
			LastTransitionTime: metav1.Now(),
		}

		apimeta.SetStatusCondition(&indexer.Status.Conditions, newCondition)

		if err := r.Status().Update(ctx, &indexer); err != nil {
			l.Error(err, "Failed to update Indexer status")
			return ctrl.Result{}, err
		}
	}

	// Requeue to ensure we re-check eventually
	return ctrl.Result{RequeueAfter: 15 * time.Minute}, nil
}

func (r *IndexerReconciler) checkHealth(ctx context.Context, indexer *torrentsv1alpha1.Indexer) (bool, string) {
	if len(indexer.Spec.Links) == 0 {
		return false, "No links defined"
	}

	// Use specific search path if available, or just the base URL
	baseURL := indexer.Spec.Links[0]
	// Remove trailing slash to avoid double slashes
	baseURL = strings.TrimRight(baseURL, "/")

	targetURL := baseURL

	// Try to find a search path
	if indexer.Spec.Search != nil && len(indexer.Spec.Search.Paths) > 0 {
		// Just take the first path
		path := indexer.Spec.Search.Paths[0].Path

		// Very naive templating/cleanup
		path = strings.ReplaceAll(path, "{{ .Keywords }}", "")
		path = strings.ReplaceAll(path, "{{ .Config.username }}", "guest") // Fallback
		path = strings.ReplaceAll(path, "{{ .Config.password }}", "guest")

		if strings.Contains(path, "{{") {
			// If complex templating remains, fallback to base URL
			targetURL = baseURL
		} else {
			if !strings.HasPrefix(path, "/") {
				targetURL = baseURL + "/" + path
			} else {
				targetURL = baseURL + path
			}
		}
	}

	if targetURL == "" {
		targetURL = baseURL
	}

	// Check if we should use FlareSolverr
	useFlareSolverr := false
	if r.FlareSolverrURL != "" {
		for _, setting := range indexer.Spec.Settings {
			if setting.Type == "info_flaresolverr" {
				useFlareSolverr = true
				break
			}
		}
	}

	// fmt.Printf("DEBUG: Indexer=%s URL=%s FS_URL='%s' UseFS=%v\n", indexer.Name, targetURL, r.FlareSolverrURL, useFlareSolverr)

	if useFlareSolverr {
		return r.checkHealthViaFlareSolverr(ctx, targetURL)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return false, fmt.Sprintf("Failed to create request: %v", err)
	}

	// Set User-Agent as some trackers require it
	req.Header.Set("User-Agent", "Prowlarr/1.0 (Text-Mode-Operator)")

	resp, err := r.HTTPClient.Do(req)
	if err != nil {
		return false, fmt.Sprintf("Connection failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, ""
	}

	return false, fmt.Sprintf("HTTP Status: %d", resp.StatusCode)
}

type flareSolverrRequest struct {
	Cmd        string `json:"cmd"`
	URL        string `json:"url"`
	MaxTimeout int    `json:"maxTimeout"`
}

type flareSolverrResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	// Solution struct {
	// 	Response string `json:"response"` // HTML content
	// } `json:"solution"`
}

func (r *IndexerReconciler) checkHealthViaFlareSolverr(ctx context.Context, url string) (bool, string) {
	reqBody := flareSolverrRequest{
		Cmd:        "request.get",
		URL:        url,
		MaxTimeout: 60000,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return false, fmt.Sprintf("Failed to marshal FS request: %v", err)
	}

	fsURL := fmt.Sprintf("%s/v1", strings.TrimRight(r.FlareSolverrURL, "/"))
	req, err := http.NewRequestWithContext(ctx, "POST", fsURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return false, fmt.Sprintf("Failed to create FS request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.HTTPClient.Do(req)
	if err != nil {
		return false, fmt.Sprintf("FlareSolverr connection failed: %v", err)
	}
	defer resp.Body.Close()

	var fsResp flareSolverrResponse
	if err := json.NewDecoder(resp.Body).Decode(&fsResp); err != nil {
		return false, fmt.Sprintf("Failed to decode FS response: %v", err)
	}

	if fsResp.Status == "ok" {
		return true, ""
	}

	return false, fmt.Sprintf("FlareSolverr Status: %s, Msg: %s", fsResp.Status, fsResp.Message)
}

// SetupWithManager sets up the controller with the Manager.
func (r *IndexerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.HTTPClient == nil {
		r.HTTPClient = &http.Client{
			Timeout: 30 * time.Second,
			// Disable redirect following since Prowlarr indexers handle redirects manually sometimes,
			// but for health check, following redirects is usually good.
			// Default Go client follows redirects.
		}
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&torrentsv1alpha1.Indexer{}).
		Complete(r)
}
