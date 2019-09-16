package controller

import (
	"github.com/utkarshmani1997/jiva-operator/pkg/controller/jivavolume"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, jivavolume.Add)
}
