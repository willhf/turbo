package turbo

import (
	"context"
	"sync"
)

// type Turbo[TModel any] struct {
// 	Model  TModel
// 	Loader *Loader[TModel] `json:"-"`
// }

// func NewConstructor[TModel any, TTurbo any](conv func(*Turbo[TModel]) TTurbo) func(models []TModel) []TTurbo {
// 	return func(models []TModel) []TTurbo {
// 		loader := &Loader[TModel]{
// 			models:   models,
// 			promises: make(map[string]*Promise[TModel, any]),
// 		}

// 		// TODO: worry about batch size here

// 		var turbos []TTurbo
// 		for _, model := range models {
// 			tb := &Turbo[TModel]{
// 				Model:  model,
// 				Loader: loader,
// 			}
// 			turbos = append(turbos, conv(tb))
// 		}

// 		return turbos
// 	}
// }

type HasLoader[TModel any] interface {
	SetLoader(loader *Loader[TModel])
	GetLoader() *Loader[TModel]
}

func Initialize[TModel HasLoader[TModel]](models []TModel) []TModel {
	loader := &Loader[TModel]{
		models:   models,
		promises: make(map[string]*Promise[TModel, any]),
	}

	// TODO: worry about batch size here

	for _, model := range models {
		model.SetLoader(loader)
	}

	return models
}

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

func LoadRelation[TModel HasLoader[TModel], TRelation any](ctx context.Context, key string, model TModel, queryFunc QueryFunc[TModel, TRelation]) (TRelation, error) {
	loader := model.GetLoader()
	var emptyResult TRelation
	loader.Lock()

	promise := loader.promises[key]
	if promise == nil {
		promise = &Promise[TModel, any]{}
		loader.promises[key] = promise
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
		promise.result = RelationLookupFunc[TModel, any](func(m TModel) any { return lookupFunc(m) })
	}

	return promise.result(model).(TRelation), nil
}

type LoadChildrenArgs[TIdentifier comparable, TModel HasLoader[TModel], TRelationModel HasLoader[TRelationModel]] struct {
	ModelIDFunc       func(TModel) TIdentifier
	QueryChildrenFunc func(context.Context, []TIdentifier) ([]TRelationModel, error)
	ParentIDFunc      func(TRelationModel) TIdentifier
}

func LoadChildren[TIdentifier comparable, TModel HasLoader[TModel], TRelationModel HasLoader[TRelationModel]](ctx context.Context, key string, model TModel, args LoadChildrenArgs[TIdentifier, TModel, TRelationModel]) ([]TRelationModel, error) {
	queryFunc := func(ctx context.Context, models []TModel) (RelationLookupFunc[TModel, any], error) {
		modelIDs := make([]TIdentifier, 0, len(models))
		for _, model := range models {
			modelIDs = append(modelIDs, args.ModelIDFunc(model))
		}
		relations, err := args.QueryChildrenFunc(ctx, modelIDs)
		if err != nil {
			return nil, err
		}
		Initialize(relations)
		grouped := make(map[TIdentifier][]TRelationModel)
		for _, relation := range relations {
			parentID := args.ParentIDFunc(relation)
			grouped[parentID] = append(grouped[parentID], relation)
		}
		return func(m TModel) any {
			return grouped[args.ModelIDFunc(m)]
		}, nil
	}
	result, err := LoadRelation(ctx, key, model, queryFunc)
	if err != nil {
		return nil, err
	}
	return result.([]TRelationModel), nil
}

// type LoadParentArgs[TIdentifier comparable, TModel any, TRelationModel any, TRelationTurbo any] struct {
// 	ModelParentIDFunc func(TModel) *TIdentifier
// 	QueryParentFunc   func(context.Context, []TIdentifier) ([]TRelationModel, error)
// 	TurboConstructor  func([]TRelationModel) []TRelationTurbo
// 	ParentIDFunc      func(TRelationTurbo) TIdentifier
// }

// func LoadParent[TIdentifier comparable, TModel any, TRelationModel any, TRelationTurbo any](ctx context.Context, loader *Loader[TModel], key string, model TModel, args LoadParentArgs[TIdentifier, TModel, TRelationModel, TRelationTurbo]) (TRelationTurbo, error) {
// 	queryFunc := func(ctx context.Context, models []TModel) (RelationLookupFunc[TModel, any], error) {
// 		uniqueParentIDs := make(map[TIdentifier]struct{})
// 		for _, model := range models {
// 			parentID := args.ModelParentIDFunc(model)
// 			if parentID != nil {
// 				uniqueParentIDs[*parentID] = struct{}{}
// 			}
// 		}
// 		parentIDs := make([]TIdentifier, 0, len(uniqueParentIDs))
// 		for parentID := range uniqueParentIDs {
// 			parentIDs = append(parentIDs, parentID)
// 		}
// 		parents, err := args.QueryParentFunc(ctx, parentIDs)
// 		if err != nil {
// 			return nil, err
// 		}
// 		turbos := args.TurboConstructor(parents)
// 		indexed := make(map[TIdentifier]TRelationTurbo)
// 		for _, turbo := range turbos {
// 			id := args.ParentIDFunc(turbo)
// 			indexed[id] = turbo
// 		}
// 		return func(m TModel) any {
// 			var zero TRelationTurbo
// 			parentID := args.ModelParentIDFunc(m)
// 			if parentID == nil {
// 				return zero
// 			}
// 			return indexed[*parentID]
// 		}, nil
// 	}
// 	result, err := LoadRelation(ctx, loader, key, model, queryFunc)
// 	if err != nil {
// 		var zero TRelationTurbo
// 		return zero, err
// 	}
// 	return result.(TRelationTurbo), nil
// }

// var authorIDs []uint
// for _, author := range authors {
// 	authorIDs = append(authorIDs, author.ID)
// }
// var books []*Book
// if err := db.Debug().Where("author_id IN (?)", authorIDs).Find(&books).Error; err != nil {
// 	return nil, err
// }
// turboBooks := NewTurboBooks(books)
// var booksByAuthorID = make(map[uint][]*TurboBook)
// for _, book := range turboBooks {
// 	tb := (*TurboBook)(book)
// 	booksByAuthorID[*book.Model.AuthorID] = append(booksByAuthorID[*book.Model.AuthorID], tb)
// }
// return func(author *Author) []*TurboBook {
// 	return booksByAuthorID[author.ID]
// }, nil
