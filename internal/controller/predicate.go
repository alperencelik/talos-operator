package controller

import (
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
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

var jobPredicate = predicate.Funcs{
	UpdateFunc: func(e event.UpdateEvent) bool {
		// Reconcile when the job is completed or failed
		oldJob, ok1 := e.ObjectOld.(*batchv1.Job)
		newJob, ok2 := e.ObjectNew.(*batchv1.Job)
		if !ok1 || !ok2 {
			return false
		}
		if oldJob.Status.Succeeded != newJob.Status.Succeeded ||
			oldJob.Status.Failed != newJob.Status.Failed {
			fmt.Println("job update status")
			return true
		}
		return false
	},
}

var TalosClusterAddonReleasePredicate = predicate.Funcs{
	UpdateFunc: func(e event.UpdateEvent) bool {
		// Only reconcile if the generation of the object has changed
		return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
	},
}
