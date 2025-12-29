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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TorrentRequestSpec defines the desired state of TorrentRequest
type TorrentRequestSpec struct {
	// Keywords to search for
	Keywords string `json:"keywords"`

	// Category to search in (e.g. "Movies", "TV")
	// +optional
	Category string `json:"category,omitempty"`

	// MinSeeders filters results by minimum seeders
	// +optional
	MinSeeders int `json:"minSeeders,omitempty"`

	// Indexers allows specifying specific indexers to use. If empty, uses all healthy public ones.
	// +optional
	Indexers []string `json:"indexers,omitempty"`
}

// TorrentRequestStatus defines the observed state of TorrentRequest
type TorrentRequestStatus struct {
	// State of the request: "Pending", "Searching", "Completed", "Failed"
	// +optional
	State string `json:"state,omitempty"`

	// FoundTorrent is the name of the Torrent CR created
	// +optional
	FoundTorrent string `json:"foundTorrent,omitempty"`

	// ResultsFound is the number of results returned by the search
	// +optional
	ResultsFound int `json:"resultsFound,omitempty"`

	// Conditions store the status conditions of the TorrentRequest
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Torrent",type="string",JSONPath=".status.foundTorrent"
// +kubebuilder:printcolumn:name="Results",type="integer",JSONPath=".status.resultsFound"
// +kubebuilder:resource:shortName=tr

// TorrentRequest is the Schema for the torrentrequests API
type TorrentRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TorrentRequestSpec   `json:"spec,omitempty"`
	Status TorrentRequestStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TorrentRequestList contains a list of TorrentRequest
type TorrentRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TorrentRequest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TorrentRequest{}, &TorrentRequestList{})
}
