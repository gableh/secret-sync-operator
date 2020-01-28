package controller

import (
	secretsync "github.com/gableh/secret-sync-operator/pkg/controller/secretsync"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, secretsync.Add)
}
