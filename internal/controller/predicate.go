package controller

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var stsPredicate = predicate.Funcs{
	UpdateFunc: func(e event.UpdateEvent) bool {
		// Only reconcile if the generation of the object has changed
		return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
	},
}

var svcPredicate = predicate.Funcs{
	UpdateFunc: func(e event.UpdateEvent) bool {
		// Only reconcile if the generation of the object has changed
		return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
	},
}

var talosMachinePredicate = predicate.Funcs{
	UpdateFunc: func(e event.UpdateEvent) bool {
		// Only reconcile if the generation of the object has changed
		return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
	},
}
