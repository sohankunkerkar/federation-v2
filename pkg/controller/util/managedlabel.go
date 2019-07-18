/*
Copyright 2019 The Kubernetes Authors.

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

package util

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	ManagedByKubeFedLabelKey   = "kubefed.k8s.io/managed"
	ManagedByKubeFedLabelValue = "true"
)

// HasManagedLabel indicates whether the given object has the managed
// label.
func HasManagedLabel(obj *unstructured.Unstructured) bool {
	labels := obj.GetLabels()
	if labels == nil {
		return false
	}
	return labels[ManagedByKubeFedLabelKey] == ManagedByKubeFedLabelValue
}

// AddManagedLabel ensures that the given object has the managed
// label.
func AddManagedLabel(obj *unstructured.Unstructured) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[ManagedByKubeFedLabelKey] = ManagedByKubeFedLabelValue
	obj.SetLabels(labels)
}

// RemoveManagedLabel ensures that the given object does not have the
// managed label.
func RemoveManagedLabel(obj *unstructured.Unstructured) {
	labels := obj.GetLabels()
	if labels == nil || labels[ManagedByKubeFedLabelKey] != ManagedByKubeFedLabelValue {
		return
	}
	delete(labels, ManagedByKubeFedLabelKey)
	obj.SetLabels(labels)
}

//CheckManagedLabel checks if the given object has the
//managed:false label
func CheckManagedLabel(obj *unstructured.Unstructured) bool {
	labels := obj.GetLabels()
	if labels == nil {
		return false
	}
	return labels[ManagedByKubeFedLabelKey] == "false"
}
