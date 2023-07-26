package utils

import (
	"context"

	"github.com/gokutils/txctx"
)

type WriterRepository[T any] interface {
	Delete(context.Context, ...T) error
	Create(context.Context, ...T) error
	Update(context.Context, ...T) error
}

type Equality[T any] func(T, T) bool

type Slice[T any] []T

func (impl Slice[T]) Filter(v T, fn Equality[T]) []T {
	ret := []T{}
	for i := range impl {
		if fn(v, impl[i]) {
			continue
		}
		ret = append(ret, v)
	}
	return ret
}

func (impl Slice[T]) Containe(v T, fn Equality[T]) bool {
	for i := range impl {
		if fn(v, impl[i]) {
			return true
		}
	}
	return false
}

type Writer[T any] struct {
	Repository WriterRepository[T]
	Equal      Equality[T]
	Current    Slice[T]
	ToUpdate   Slice[T]
	ToCreate   Slice[T]
	ToDelete   Slice[T]
}

func (impl *Writer[T]) Restore(value T) {
	impl.Current = append(impl.Current, value)
	impl.ToDelete = impl.ToDelete.Filter(value, impl.Equal)
}

func (impl *Writer[T]) Delete(values ...T) {
	for i := range values {
		if impl.Current.Containe(values[i], impl.Equal) {
			impl.Current = impl.Current.Filter(values[i], impl.Equal)
			impl.ToDelete = append(impl.ToDelete, values[i])
		}
		if impl.ToUpdate.Containe(values[i], impl.Equal) {
			impl.Current = impl.ToUpdate.Filter(values[i], impl.Equal)
			impl.ToDelete = append(impl.ToDelete, values[i])
		}
		if impl.ToCreate.Containe(values[i], impl.Equal) {
			impl.ToCreate = impl.ToCreate.Filter(values[i], impl.Equal)
		}
	}
}
func (impl *Writer[T]) SetToUpdate(v T) {
	if impl.Current.Containe(v, impl.Equal) {
		impl.Current = impl.Current.Filter(v, impl.Equal)
		impl.ToUpdate = append(impl.ToUpdate, v)
	} else if impl.ToDelete.Containe(v, impl.Equal) {
		impl.Current = impl.ToDelete.Filter(v, impl.Equal)
		impl.ToUpdate = append(impl.ToUpdate, v)
	}
}

func (impl *Writer[T]) GetAllActive() []T {
	return append(append(impl.Current, impl.ToCreate...), impl.ToUpdate...)
}

func (impl *Writer[T]) AddToCreate(value ...T) {
	impl.ToCreate = append(impl.ToCreate, value...)
}

func (impl *Writer[T]) Search(c func(value T) bool) T {
	for i := range impl.Current {
		if c(impl.Current[i]) {
			return impl.Current[i]
		}
	}
	for i := range impl.ToCreate {
		if c(impl.ToCreate[i]) {
			return impl.ToCreate[i]
		}
	}
	for i := range impl.ToUpdate {
		if c(impl.ToUpdate[i]) {
			return impl.ToUpdate[i]
		}
	}
	var noop T
	return noop
}

func (impl *Writer[T]) Save(ctx context.Context) error {
	if err := impl.Repository.Delete(ctx, impl.ToDelete...); err != nil {
		return err
	}
	if err := impl.Repository.Create(ctx, impl.ToCreate...); err != nil {
		return err
	}
	if err := impl.Repository.Update(ctx, impl.ToUpdate...); err != nil {
		return err
	}
	if txctx.IsTxContext(ctx) {
		txctx.Add(ctx, impl)
	} else {
		impl.Commit(ctx)
	}
	return nil
}

func (impl *Writer[T]) Commit(ctx context.Context) error {
	impl.Current = append(append(impl.Current, impl.ToCreate...), impl.ToUpdate...)
	impl.ToCreate = []T{}
	impl.ToDelete = []T{}
	impl.ToUpdate = []T{}
	return nil
}

func (impl *Writer[T]) Rollback(ctx context.Context) error {
	return nil
}

func NewWriter[T any](repository WriterRepository[T], equalFn Equality[T]) *Writer[T] {
	return &Writer[T]{
		Equal:      equalFn,
		Repository: repository,
	}
}
