/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

// Package v1alpha1 contains API Schema definitions for the isindir v1alpha1 API group
// +kubebuilder:object:generate=true
// +groupName=isindir.github.com
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "isindir.github.com", Version: "v1alpha1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = runtime.NewSchemeBuilder(addToScheme)

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

// addToScheme adds the SopsSecret types for this group-version to the scheme.
func addToScheme(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion, &SopsSecret{}, &SopsSecretList{})
	metav1.AddToGroupVersion(scheme, GroupVersion)
	return nil
}
