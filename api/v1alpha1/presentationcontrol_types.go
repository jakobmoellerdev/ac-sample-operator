/*
Copyright 2022.

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
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

// PresentationControlSpec defines the desired state of PresentationControl
type PresentationControlSpec struct {
	Formula     string      `json:"formula"`
	Parameters  Parameters  `json:"parameters,omitempty"`
	Recalculate Recalculate `json:"recalculate,omitempty"`
}

type Recalculate struct {
	Every string `json:"every,omitempty"`
}

type Parameters map[string]Parameter

func (p Parameters) String() string {
	outs := make([]string, 0)
	for k, v := range p {
		outs = append(outs, fmt.Sprintf("%s(%s):%s", k, v.Type, v.Value))
	}
	return fmt.Sprintf("[%s]", strings.Join(outs, ", "))
}

type ParameterType string

const (
	ParameterTypeNumber = "number"
	ParameterTypeString = "string"
	ParameterTypeSecret = "secret"
)

type Parameter struct {
	Value string        `json:"value"`
	Type  ParameterType `json:"type,omitempty"`
}

// PresentationControlStatus defines the observed state of PresentationControl
type PresentationControlStatus struct {
	Result             string `json:"result"`
	ObservedGeneration int64  `json:"observedGeneration"`
	ObservedAt         string `json:"observedAt"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Formula",type=string,JSONPath=".spec.formula"
//+kubebuilder:printcolumn:name="Result",type=string,JSONPath=".status.result"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// PresentationControl is the Schema for the presentationcontrols API
type PresentationControl struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PresentationControlSpec   `json:"spec,omitempty"`
	Status PresentationControlStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PresentationControlList contains a list of PresentationControl
type PresentationControlList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PresentationControl `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PresentationControl{}, &PresentationControlList{})
}
