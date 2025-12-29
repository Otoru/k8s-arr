package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	torrentsv1alpha1 "vitoru.fun/torrents/api/v1alpha1"
	"vitoru.fun/torrents/internal/parser"
)

// TorrentRequestReconciler reconciles a TorrentRequest object
type TorrentRequestReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	HTTPClient      *http.Client
	FlareSolverrURL string
}

// +kubebuilder:rbac:groups=torrents.vitoru.fun,resources=torrentrequests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=torrents.vitoru.fun,resources=torrentrequests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=torrents.vitoru.fun,resources=torrents,verbs=get;list;watch;create;update;patch;delete

func (r *TorrentRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	var tr torrentsv1alpha1.TorrentRequest
	if err := r.Get(ctx, req.NamespacedName, &tr); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// If already completed or failed, stop
	if tr.Status.State == "Completed" || tr.Status.State == "Failed" {
		return ctrl.Result{}, nil
	}

	// Update state to Searching if Pending/Empty
	if tr.Status.State == "" || tr.Status.State == "Pending" {
		tr.Status.State = "Searching"
		meta.SetStatusCondition(&tr.Status.Conditions, metav1.Condition{
			Type:    "Ready",
			Status:  metav1.ConditionFalse,
			Reason:  "Searching",
			Message: "Searching for torrents across indexers",
		})
		if err := r.Status().Update(ctx, &tr); err != nil {
			return ctrl.Result{}, err
		}
		// Requeue to process search immediately with new state
		return ctrl.Result{Requeue: true}, nil
	}

	l.Info("Starting search for torrent", "keywords", tr.Spec.Keywords)

	// List Indexers
	var indexerList torrentsv1alpha1.IndexerList
	if err := r.List(ctx, &indexerList); err != nil {
		l.Error(err, "Failed to list indexers")
		return ctrl.Result{}, err
	}

	var allResults []parser.ParseResult

	// Iterate and search
	for _, indexer := range indexerList.Items {
		// Skip unhealthy
		if !indexer.Status.Healthy {
			continue
		}
		// Skip if user requested specific indexers
		if len(tr.Spec.Indexers) > 0 {
			found := false
			for _, name := range tr.Spec.Indexers {
				if name == indexer.Name {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		l.Info("Querying indexer", "name", indexer.Name)
		results, err := r.searchIndexer(ctx, &indexer, tr.Spec.Keywords)
		if err != nil {
			l.Error(err, "Search failed for indexer", "name", indexer.Name)
			torrentSearchesTotal.WithLabelValues(indexer.Name, "failed").Inc()
			continue
		}

		status := "empty"
		if len(results) > 0 {
			status = "success"
		}
		torrentSearchesTotal.WithLabelValues(indexer.Name, status).Inc()

		l.Info("Found results", "indexer", indexer.Name, "count", len(results))
		// Enrich results with indexer name
		for i := range results {
			// Quick fix to ensure indexer name is populated if parser didn't do it
			if results[i].Indexer == "" {
				results[i].Indexer = indexer.Name
			}
		}
		allResults = append(allResults, results...)
	}

	tr.Status.ResultsFound = len(allResults)

	if len(allResults) == 0 {
		tr.Status.State = "Failed"
		meta.SetStatusCondition(&tr.Status.Conditions, metav1.Condition{
			Type:    "Ready",
			Status:  metav1.ConditionFalse,
			Reason:  "NotFound",
			Message: "No results found across all indexers",
		})
		l.Info("No results found across all indexers")

		// Record Duration and Failure Count
		duration := time.Since(tr.CreationTimestamp.Time).Seconds()
		torrentRequestFailureDuration.Observe(duration)
		torrentRequestsFailedTotal.Inc()

		if err := r.Status().Update(ctx, &tr); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Sort and Pick Best
	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].Seeders > allResults[j].Seeders
	})

	var bestTorrent *parser.ParseResult
	for i := range allResults {
		if allResults[i].Seeders >= tr.Spec.MinSeeders {
			bestTorrent = &allResults[i]
			break
		}
	}

	if bestTorrent == nil {
		if tr.Spec.MinSeeders > 0 {
			l.Info("No results met minSeeders requirement")
			tr.Status.State = "Failed"
			r.Status().Update(ctx, &tr)
			return ctrl.Result{}, nil
		}
		bestTorrent = &allResults[0]
	}

	// Create Torrent CR
	safeName := strings.ToLower(strings.ReplaceAll(bestTorrent.Title, " ", "-"))
	safeName = strings.ReplaceAll(safeName, ".", "-")
	reg, _ := regexp.Compile("[^a-z0-9-]+")
	safeName = reg.ReplaceAllString(safeName, "")
	if len(safeName) > 50 {
		safeName = safeName[:50]
	}
	safeName = fmt.Sprintf("%s-%d", safeName, time.Now().Unix())

	torrentCR := &torrentsv1alpha1.Torrent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      safeName,
			Namespace: tr.Namespace,
			Labels: map[string]string{
				"created-by": tr.Name,
			},
		},
		Spec: torrentsv1alpha1.TorrentSpec{
			Title:    bestTorrent.Title,
			Magnet:   bestTorrent.Magnet,
			Size:     bestTorrent.Size,
			Seeders:  bestTorrent.Seeders,
			Leechers: bestTorrent.Leechers,
			Indexer:  bestTorrent.Indexer,
		},
	}

	// Set OwnerReference
	if err := ctrl.SetControllerReference(&tr, torrentCR, r.Scheme); err != nil {
		l.Error(err, "Failed to set controller reference")
		return ctrl.Result{}, err
	}

	if err := r.Create(ctx, torrentCR); err != nil {
		l.Error(err, "Failed to create Torrent CR")
		return ctrl.Result{}, err
	}

	// Update Request Status
	tr.Status.State = "Completed"
	tr.Status.FoundTorrent = safeName
	meta.SetStatusCondition(&tr.Status.Conditions, metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionTrue,
		Reason:  "Found",
		Message: fmt.Sprintf("Found torrent: %s", safeName),
	})

	// Record Duration
	duration := time.Since(tr.CreationTimestamp.Time).Seconds()
	torrentRequestDuration.WithLabelValues(bestTorrent.Indexer).Observe(duration)
	torrentsCreatedTotal.WithLabelValues(bestTorrent.Indexer).Inc()

	if err := r.Status().Update(ctx, &tr); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *TorrentRequestReconciler) searchIndexer(ctx context.Context, indexer *torrentsv1alpha1.Indexer, keywords string) ([]parser.ParseResult, error) {
	l := log.FromContext(ctx)
	// Construct URL (Reuse logic from checkHealth ideally, but we need parameter injection)
	if len(indexer.Spec.Links) == 0 {
		return nil, fmt.Errorf("no links")
	}
	baseURL := strings.TrimRight(indexer.Spec.Links[0], "/")
	targetURL := baseURL

	if indexer.Spec.Search != nil && len(indexer.Spec.Search.Paths) > 0 {
		pathTmpl := indexer.Spec.Search.Paths[0].Path

		// Prepare template data
		data := struct {
			Keywords string
			Config   map[string]string
		}{
			Keywords: url.QueryEscape(keywords),
			Config: map[string]string{
				"username": "guest",
				"password": "guest",
			},
		}

		funcMap := template.FuncMap{
			"re_replace": func(input, pattern, replacement string) string {
				re, err := regexp.Compile(pattern)
				if err != nil {
					return input
				}
				return re.ReplaceAllString(input, replacement)
			},
			"replace": func(input, from, to string) string {
				return strings.ReplaceAll(input, from, to)
			},
		}

		// Parse and Execute
		t, err := template.New("path").Funcs(funcMap).Parse(pathTmpl)
		if err == nil {
			var buf bytes.Buffer
			if err := t.Execute(&buf, data); err == nil {
				if !strings.HasPrefix(buf.String(), "/") {
					targetURL = baseURL + "/" + buf.String()
				} else {
					targetURL = baseURL + buf.String()
				}
			} else {
				l.Error(err, "Template execute failed, falling back")
				targetURL = baseURL
			}
		} else {
			l.Error(err, "Template parse failed")
			targetURL = baseURL
		}
	}
	fmt.Printf("DEBUG: Searching Indexer %s URL: %s\n", indexer.Name, targetURL)

	var body []byte
	var err error

	useFlareSolverr := false
	if r.FlareSolverrURL != "" {
		for _, setting := range indexer.Spec.Settings {
			if setting.Type == "info_flaresolverr" {
				useFlareSolverr = true
				break
			}
		}
	}

	if useFlareSolverr {
		body, err = r.doRequestFS(ctx, targetURL)
	} else {
		req, _ := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
		req.Header.Set("User-Agent", "Prowlarr/1.0 (Text-Mode-Operator)")
		resp, er := r.HTTPClient.Do(req)
		if er != nil {
			return nil, er
		}
		defer resp.Body.Close()
		body, err = io.ReadAll(resp.Body)
	}

	if err != nil {
		return nil, err
	}

	return parser.ParseHTML(string(body), indexer)
}

func (r *TorrentRequestReconciler) doRequestFS(ctx context.Context, targetURL string) ([]byte, error) {
	reqBody := map[string]interface{}{
		"cmd":        "request.get",
		"url":        targetURL,
		"maxTimeout": 60000,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal FS request: %v", err)
	}

	fsURL := fmt.Sprintf("%s/v1", strings.TrimRight(r.FlareSolverrURL, "/"))
	req, err := http.NewRequestWithContext(ctx, "POST", fsURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create FS request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("FlareSolverr connection failed: %v", err)
	}
	defer resp.Body.Close()

	var fsResp struct {
		Status   string `json:"status"`
		Solution struct {
			Response string `json:"response"` // HTML content
		} `json:"solution"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&fsResp); err != nil {
		return nil, fmt.Errorf("failed to decode FS response: %v", err)
	}

	if fsResp.Status == "ok" {
		return []byte(fsResp.Solution.Response), nil
	}

	return nil, fmt.Errorf("FlareSolverr failed: %s", fsResp.Status)
}

func (r *TorrentRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&torrentsv1alpha1.TorrentRequest{}).
		Complete(r)
}
