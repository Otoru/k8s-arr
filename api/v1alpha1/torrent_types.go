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

// TorrentSpec defines the desired state of Torrent
type TorrentSpec struct {
	// Title of the torrent release
	Title string `json:"title"`

	// Magnet link
	Magnet string `json:"magnet"`

	// InfoHash of the torrent (optional, can be extracted from magnet)
	// +optional
	InfoHash string `json:"infoHash,omitempty"`

	// Size of the content (string representation, e.g., "1.5 GB")
	// +optional
	Size string `json:"size,omitempty"`

	// Seeders count at time of discovery
	// +optional
	Seeders int `json:"seeders,omitempty"`

	// Leechers count at time of discovery
	// +optional
	Leechers int `json:"leechers,omitempty"`

	// Indexer that provided this torrent
	// +optional
	Indexer string `json:"indexer,omitempty"`

	// PublishedAt is when the torrent was uploaded
	// +optional
	PublishedAt *metav1.Time `json:"publishedAt,omitempty"`
}

// TorrentStatus defines the observed state of Torrent
type TorrentStatus struct {
	// Insert additional status field - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Torrent is the Schema for the torrents API
type Torrent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TorrentSpec   `json:"spec,omitempty"`
	Status TorrentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TorrentList contains a list of Torrent
type TorrentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Torrent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Torrent{}, &TorrentList{})
}
