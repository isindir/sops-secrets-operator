package controllers

import (
	"reflect"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type SecretDataTypeChangedPredicate struct {
	predicate.Funcs
}

func (d SecretDataTypeChangedPredicate) Update(e event.UpdateEvent) bool {
	oldSecret, oldOK := e.ObjectOld.(*corev1.Secret)
	newSecret, newOK := e.ObjectNew.(*corev1.Secret)

	if !oldOK && !newOK {
		return false
	}

	if !reflect.DeepEqual(oldSecret.Data, newSecret.Data) {
		return true
	}

	if !reflect.DeepEqual(oldSecret.StringData, newSecret.StringData) {
		return true
	}

	if oldSecret.Type != newSecret.Type {
		return true
	}

	return false
}
