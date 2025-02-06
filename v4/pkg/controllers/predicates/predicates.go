package predicates

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func OnlyDelete() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc:  never(event.CreateEvent{}),
		UpdateFunc:  never(event.UpdateEvent{}),
		DeleteFunc:  always(event.DeleteEvent{}),
		GenericFunc: never(event.GenericEvent{}),
	}
}

func OnlyCreate() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc:  always(event.CreateEvent{}),
		UpdateFunc:  never(event.UpdateEvent{}),
		DeleteFunc:  never(event.DeleteEvent{}),
		GenericFunc: never(event.GenericEvent{}),
	}
}

func OnlyUpdate() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc:  never(event.CreateEvent{}),
		UpdateFunc:  always(event.UpdateEvent{}),
		DeleteFunc:  never(event.DeleteEvent{}),
		GenericFunc: never(event.GenericEvent{}),
	}
}

func OnlyCreateUpdate() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc:  always(event.CreateEvent{}),
		UpdateFunc:  always(event.UpdateEvent{}),
		DeleteFunc:  never(event.DeleteEvent{}),
		GenericFunc: never(event.GenericEvent{}),
	}
}

func never[t any](_ t) func(e t) bool {
	return func(e t) bool {
		return false
	}
}

func always[t any](_ t) func(e t) bool {
	return func(e t) bool {
		return true
	}
}
