package turbo

import (
	"context"
	"sync"
)

type Turbo[TModel any] struct {
	Model  TModel
	Loader *Loader[TModel] `json:"-"`
}

func (t *Turbo[TModel]) GetModel() TModel {
	return t.Model
}

func NewConstructor[TModel any, TTurbo any](conv func(*Turbo[TModel]) TTurbo) func(models []TModel) []TTurbo {
	return func(models []TModel) []TTurbo {
		loader := &Loader[TModel]{
			models:   models,
			promises: make(map[string]*Promise[TModel, any]),
		}

		// TODO: worry about batch size here

		var turbos []TTurbo
		for _, model := range models {
			tb := &Turbo[TModel]{
				Model:  model,
				Loader: loader,
			}
			turbos = append(turbos, conv(tb))
		}

		return turbos
	}
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

func LoadRelation[TModel any, TRelation any](ctx context.Context, loader *Loader[TModel], key string, model TModel, queryFunc QueryFunc[TModel, TRelation]) (TRelation, error) {
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

type LoadChildrenArgs[TModel any, TRelationModel any, TRelationTurbo any] struct {
	ModelIDFunc       func(TModel) uint
	QueryChildrenFunc func(context.Context, []uint) ([]TRelationModel, error)
	TurboConstructor  func([]TRelationModel) []TRelationTurbo
	ParentIDFunc      func(TRelationTurbo) uint
}

func LoadChildren[TModel any, TRelationModel any, TRelationTurbo any](ctx context.Context, loader *Loader[TModel], key string, model TModel, args LoadChildrenArgs[TModel, TRelationModel, TRelationTurbo]) ([]TRelationTurbo, error) {
	queryFunc := func(ctx context.Context, models []TModel) (RelationLookupFunc[TModel, any], error) {
		modelIDs := make([]uint, 0, len(models))
		for _, model := range models {
			modelIDs = append(modelIDs, args.ModelIDFunc(model))
		}
		children, err := args.QueryChildrenFunc(ctx, modelIDs)
		if err != nil {
			return nil, err
		}
		turbos := args.TurboConstructor(children)
		grouped := make(map[uint][]TRelationTurbo)
		for _, turbo := range turbos {
			parentID := args.ParentIDFunc(turbo)
			grouped[parentID] = append(grouped[parentID], turbo)
		}
		return func(m TModel) any {
			return grouped[args.ModelIDFunc(m)]
		}, nil
	}
	result, err := LoadRelation(ctx, loader, key, model, queryFunc)
	if err != nil {
		return nil, err
	}
	return result.([]TRelationTurbo), nil
}

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
