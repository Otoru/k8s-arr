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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// IndexerSpec defines the desired state of Indexer
type IndexerSpec struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"description,omitempty"`
	Language        string   `json:"language"`
	Type            string   `json:"type"`
	Encoding        string   `json:"encoding,omitempty"`
	Links           []string `json:"links"`
	LegacyLinks     []string `json:"legacylinks,omitempty"`
	Certificates    []string `json:"certificates,omitempty"`
	RequestDelay    string   `json:"requestDelay,omitempty"`
	FollowRedirect  bool     `json:"followredirect,omitempty"`
	TestLinkTorrent bool     `json:"testlinktorrent,omitempty"`

	Caps     Caps            `json:"caps"`
	Settings []SettingsField `json:"settings,omitempty"`
	Login    *Login          `json:"login,omitempty"`
	Search   *Search         `json:"search,omitempty"`
	Download *DownloadBlock  `json:"download,omitempty"`
}

type Caps struct {
	Categories       map[string]string `json:"categories,omitempty"`
	CategoryMappings []CategoryMapping `json:"categorymappings,omitempty"`
	Modes            Modes             `json:"modes"`
	AllowRawSearch   bool              `json:"allowrawsearch,omitempty"`
}

type CategoryMapping struct {
	ID      string `json:"id"`
	Cat     string `json:"cat"`
	Desc    string `json:"desc,omitempty"`
	Default bool   `json:"default,omitempty"`
}

type Modes struct {
	Search      []string `json:"search"`
	TvSearch    []string `json:"tv-search,omitempty"`
	MovieSearch []string `json:"movie-search,omitempty"`
	MusicSearch []string `json:"music-search,omitempty"`
	BookSearch  []string `json:"book-search,omitempty"`
}

type SettingsField struct {
	Name     string            `json:"name"`
	Label    string            `json:"label,omitempty"`
	Type     string            `json:"type"`
	Default  string            `json:"default,omitempty"`
	Options  map[string]string `json:"options,omitempty"`
	Defaults []string          `json:"defaults,omitempty"`
}

type Login struct {
	Method            string                   `json:"method,omitempty"`
	Cookies           []string                 `json:"cookies,omitempty"`
	Path              string                   `json:"path,omitempty"`
	SubmitPath        string                   `json:"submitpath,omitempty"`
	Form              string                   `json:"form,omitempty"`
	Captcha           *CaptchaBlock            `json:"captcha,omitempty"`
	Inputs            map[string]string        `json:"inputs,omitempty"`
	Selectors         bool                     `json:"selectors,omitempty"`
	SelectorInputs    map[string]SelectorBlock `json:"selectorinputs,omitempty"`
	GetSelectorInputs map[string]SelectorBlock `json:"getselectorinputs,omitempty"`
	Error             []ErrorBlock             `json:"error,omitempty"`
	Test              *PageTestBlock           `json:"test,omitempty"`
	Headers           map[string][]string      `json:"headers,omitempty"`
}

type PageTestBlock struct {
	Path     string `json:"path"`
	Selector string `json:"selector,omitempty"`
}

type CaptchaBlock struct {
	Type     string `json:"type"`
	Selector string `json:"selector"`
	Input    string `json:"input"`
}

type ErrorBlock struct {
	Path     string        `json:"path,omitempty"`
	Selector string        `json:"selector"`
	Message  SelectorBlock `json:"message,omitempty"`
}

type SelectorBlock struct {
	Selector  string            `json:"selector,omitempty"`
	Attribute string            `json:"attribute,omitempty"`
	Optional  bool              `json:"optional,omitempty"`
	Default   string            `json:"default,omitempty"`
	Case      map[string]string `json:"case,omitempty"`
	Remove    string            `json:"remove,omitempty"`
	Text      string            `json:"text,omitempty"`
	Filters   []FilterBlock     `json:"filters,omitempty"`
}

type FilterBlock struct {
	Name string   `json:"name"`
	Args []string `json:"args,omitempty"`
}

type Search struct {
	Path                 string              `json:"path,omitempty"`
	Paths                []SearchPathBlock   `json:"paths,omitempty"`
	AllowEmptyInputs     bool                `json:"allowEmptyInputs,omitempty"`
	Inputs               map[string]string   `json:"inputs,omitempty"`
	Headers              map[string][]string `json:"headers,omitempty"`
	KeywordsFilters      []FilterBlock       `json:"keywordsfilters,omitempty"`
	Error                []ErrorBlock        `json:"error,omitempty"`
	PreprocessingFilters []FilterBlock       `json:"preprocessingfilters,omitempty"`
	Rows                 RowsBlock           `json:"rows"`
	Fields               FieldsBlock         `json:"fields"`
}

type SearchPathBlock struct {
	Path           string            `json:"path"`
	Method         string            `json:"method,omitempty"`
	FollowRedirect bool              `json:"followredirect,omitempty"`
	Categories     []string          `json:"categories,omitempty"`
	Inputs         map[string]string `json:"inputs,omitempty"`
	InheritInputs  bool              `json:"inheritinputs,omitempty"`
	QuerySeparator string            `json:"queryseparator,omitempty"`
	Response       *ResponseBlock    `json:"response,omitempty"`
}

type ResponseBlock struct {
	Type             string `json:"type"`
	NoResultsMessage string `json:"noResultsMessage,omitempty"`
}

type RowsBlock struct {
	After                           int               `json:"after,omitempty"`
	DateHeaders                     SelectorBlock     `json:"dateheaders,omitempty"`
	Selector                        string            `json:"selector,omitempty"`
	Attribute                       string            `json:"attribute,omitempty"`
	Optional                        bool              `json:"optional,omitempty"`
	Multiple                        bool              `json:"multiple,omitempty"`
	MissingAttributeEqualsNoResults bool              `json:"missingAttributeEqualsNoResults,omitempty"`
	Case                            map[string]string `json:"case,omitempty"`
	Remove                          string            `json:"remove,omitempty"`
	Text                            string            `json:"text,omitempty"`
	Filters                         []RowFilterBlock  `json:"filters,omitempty"`
	Count                           SelectorBlock     `json:"count,omitempty"`
}

type RowFilterBlock struct {
	Name string   `json:"name"`
	Args []string `json:"args,omitempty"`
}

// FieldsBlock is a map of selector blocks, but we can't map[string]SelectorBlock easily in CRD if keys are dynamic.
// However, the keys are somewhat standard (title, seeders, etc.) but can have suffixes.
// For now, we'll use runtime.RawExtension or just map[string]SelectorBlock and hope controller-gen handles it or we use +kubebuilder:pruning:PreserveUnknownFields
// Actually, map[string]SelectorBlock usually works.
type FieldsBlock map[string]SelectorBlock

type DownloadBlock struct {
	Method    string              `json:"method,omitempty"`
	Before    *BeforeBlock        `json:"before,omitempty"`
	Selectors []SelectorBlock     `json:"selectors,omitempty"`
	InfoHash  *InfoHashBlock      `json:"infohash,omitempty"`
	Headers   map[string][]string `json:"headers,omitempty"`
}

type BeforeBlock struct {
	Path         string            `json:"path,omitempty"`
	PathSelector *SelectorField    `json:"pathselector,omitempty"`
	Method       string            `json:"method,omitempty"`
	Inputs       map[string]string `json:"inputs,omitempty"`
}

type SelectorField struct {
	Text string `json:"text,omitempty"`
}

type InfoHashBlock struct {
	Hash string `json:"hash,omitempty"`
	Type string `json:"type,omitempty"`
}

// IndexerStatus defines the observed state of Indexer
type IndexerStatus struct {
	// Healthy indicates if the indexer is considered healthy.
	Healthy bool `json:"healthy"`

	// LastHealthCheckTime is the last time the health check was performed.
	// +optional
	LastHealthCheckTime *metav1.Time `json:"lastHealthCheckTime,omitempty"`

	// ErrorMessage captures the error from the last failed health check.
	// +optional
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Healthy",type="boolean",JSONPath=".status.healthy"
// +kubebuilder:printcolumn:name="Last Check",type="date",JSONPath=".status.lastHealthCheckTime"
// +kubebuilder:printcolumn:name="Error",type="string",JSONPath=".status.errorMessage"

// Indexer is the Schema for the indexers API
type Indexer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IndexerSpec   `json:"spec,omitempty"`
	Status IndexerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IndexerList contains a list of Indexer
type IndexerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Indexer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Indexer{}, &IndexerList{})
}
