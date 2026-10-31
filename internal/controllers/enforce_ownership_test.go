/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

package controllers

import (
	"testing"

	isindirv1alpha3 "github.com/isindir/sops-secrets-operator/api/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestShouldEnforceOwnership(t *testing.T) {
	tests := []struct {
		name                    string
		defaultEnforceOwnership bool
		specEnforceOwnership    *bool
		expectedResult          bool
	}{
		{
			name:                    "Default false, spec nil - should return false",
			defaultEnforceOwnership: false,
			specEnforceOwnership:    nil,
			expectedResult:          false,
		},
		{
			name:                    "Default true, spec nil - should return true",
			defaultEnforceOwnership: true,
			specEnforceOwnership:    nil,
			expectedResult:          true,
		},
		{
			name:                    "Default false, spec true - should return true (spec overrides)",
			defaultEnforceOwnership: false,
			specEnforceOwnership:    ptr.To(true),
			expectedResult:          true,
		},
		{
			name:                    "Default true, spec false - should return false (spec overrides)",
			defaultEnforceOwnership: true,
			specEnforceOwnership:    ptr.To(false),
			expectedResult:          false,
		},
		{
			name:                    "Default false, spec false - should return false",
			defaultEnforceOwnership: false,
			specEnforceOwnership:    ptr.To(false),
			expectedResult:          false,
		},
		{
			name:                    "Default true, spec true - should return true",
			defaultEnforceOwnership: true,
			specEnforceOwnership:    ptr.To(true),
			expectedResult:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reconciler := &SopsSecretReconciler{
				DefaultEnforceOwnership: tt.defaultEnforceOwnership,
			}

			sopsSecret := &isindirv1alpha3.SopsSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-sopssecret",
					Namespace: "default",
				},
				Spec: isindirv1alpha3.SopsSecretSpec{
					EnforceOwnership: tt.specEnforceOwnership,
					SecretsTemplate: []isindirv1alpha3.SopsSecretTemplate{
						{Name: "test-secret"},
					},
				},
			}

			result := reconciler.shouldEnforceOwnership(sopsSecret)

			if result != tt.expectedResult {
				t.Errorf("shouldEnforceOwnership() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}
