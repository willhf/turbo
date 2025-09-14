package turbo

import (
	"context"
	"sync"
)

type RelationLookupFunc[TModel any, TRelation any] func(TModel) TRelation

type QueryFunc[TModel any, TRelation any] func(context.Context, []TModel) (RelationLookupFunc[TModel, TRelation], error)

type Promise[TModel any, TResult any] struct {
	sync.Mutex

	err    error
	result RelationLookupFunc[TModel, TResult]
}

type Loader[TModel any] struct {
	sync.Mutex
	models   []TModel
	promises map[string]*Promise[TModel, any]
}

func LoadRelation[TModel any, TRelation any](ctx context.Context, loader *Loader[TModel], key string, model TModel, queryFunc QueryFunc[TModel, TRelation]) (TRelation, error) {
	var emptyResult TRelation
	loader.Lock()

	promise := loader.promises[key]
	if promise == nil {
		promise = &Promise[TModel, any]{}
	}

	promise.Lock()
	defer promise.Unlock()

	loader.Unlock()

	if promise.err != nil {
		return emptyResult, promise.err
	}

	if promise.result == nil {
		lookupFunc, err := queryFunc(ctx, loader.models)
		if err != nil {
			promise.err = err
			return emptyResult, err
		}
		promise.result = lookupFunc.(RelationLookupFunc[TModel, TRelation])
	}

	return promise.result(model).(TRelation), nil
}
