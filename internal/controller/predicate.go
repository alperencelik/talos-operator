package controller

import (
	batchv1 "k8s.io/api/batch/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// generationChangedPredicate returns a predicate that only triggers on generation changes.
func generationChangedPredicate() predicate.Funcs {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
		},
	}
}

var stsPredicate = generationChangedPredicate()

var svcPredicate = generationChangedPredicate()

var talosMachinePredicate = generationChangedPredicate()

var jobPredicate = predicate.Funcs{
	UpdateFunc: func(e event.UpdateEvent) bool {
		oldJob, ok1 := e.ObjectOld.(*batchv1.Job)
		newJob, ok2 := e.ObjectNew.(*batchv1.Job)
		if !ok1 || !ok2 {
			return false
		}
		return oldJob.Status.Succeeded != newJob.Status.Succeeded ||
			oldJob.Status.Failed != newJob.Status.Failed
	},
}

var TalosClusterAddonReleasePredicate = generationChangedPredicate()
