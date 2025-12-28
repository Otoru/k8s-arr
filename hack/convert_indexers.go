package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"

	// Import the API package to use the structs.
	// We need to use the full module path.
	"vitoru.fun/torrents/api/v1alpha1"
)

func main() {
	inputDir := "../Indexers/definitions/v11"
	outputDir := "examples"

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		panic(err)
	}

	files, err := ioutil.ReadDir(inputDir)
	if err != nil {
		panic(err)
	}

	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	serializer := json.NewSerializerWithOptions(json.DefaultMetaFactory, scheme, scheme, json.SerializerOptions{Yaml: true})

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".yml" {
			continue
		}

		content, err := ioutil.ReadFile(filepath.Join(inputDir, file.Name()))
		if err != nil {
			fmt.Printf("Error reading %s: %v\n", file.Name(), err)
			continue
		}

		var spec v1alpha1.IndexerSpec
		if err := yaml.Unmarshal(content, &spec); err != nil {
			fmt.Printf("Error unmarshalling %s: %v\n", file.Name(), err)
			continue
		}

		if spec.Type != "public" {
			continue
		}

		// Filter out FlareSolverr settings
		// Validation: We keep them now as per user request
		// var cleanSettings []v1alpha1.SettingsField
		// for _, s := range spec.Settings {
		// 	if s.Type == "info_flaresolverr" || s.Name == "info_flaresolverr" {
		// 		continue
		// 	}
		// 	cleanSettings = append(cleanSettings, s)
		// }
		// spec.Settings = cleanSettings

		indexer := &IndexerNoStatus{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "torrents.vitoru.fun/v1alpha1",
				Kind:       "Indexer",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: strings.ToLower(strings.ReplaceAll(spec.Name, " ", "-")),
			},
			Spec: spec,
		}

		// Sanitize name for K8s (lowercase, alphanumeric, -, .)
		indexer.Name = strings.ToLower(indexer.Name)
		indexer.Name = strings.ReplaceAll(indexer.Name, "_", "-")
		indexer.Name = strings.ReplaceAll(indexer.Name, ".", "-")
		// Remove parentheses which caused issues before
		indexer.Name = strings.ReplaceAll(indexer.Name, "(", "")
		indexer.Name = strings.ReplaceAll(indexer.Name, ")", "")

		outputFile := filepath.Join(outputDir, strings.TrimSuffix(file.Name(), ".yml")+".yaml")
		f, err := os.Create(outputFile)
		if err != nil {
			fmt.Printf("Error creating %s: %v\n", outputFile, err)
			continue
		}

		if err := serializer.Encode(indexer, f); err != nil {
			fmt.Printf("Error encoding %s: %v\n", outputFile, err)
			f.Close()
			continue
		}
		f.Close()
		fmt.Printf("Check generated %s\n", outputFile)
	}
}

type IndexerNoStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              v1alpha1.IndexerSpec `json:"spec,omitempty"`
}

func (in *IndexerNoStatus) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(IndexerNoStatus)
	*out = *in
	out.Spec = in.Spec // Shallow copy is enough for serialization
	return out
}
