package controller

import (
	"github.com/isindir/sops-secrets-operator/pkg/controller/sopssecret"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, sopssecret.Add)
}
