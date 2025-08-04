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

// var jobPredicate = predicate.Funcs{
// GenericFunc: func(e event.GenericEvent) bool {
// // Only reconcile if the job is not completed or failed
// job, ok := e.Object.(*batchv1.Job)
// if !ok {
// return false
// }
// isFinished := func(j *batchv1.Job) bool {
// return j.Status.Succeeded > 0 || j.Status.Failed > 0
// }
// return isFinished(job)
// },
// }
