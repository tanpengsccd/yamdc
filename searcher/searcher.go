package searcher

import (
	"context"
	"yamdc/model"
)

type ISearcher interface {
	Name() string
	Search(ctx context.Context, number *model.Number) (*model.AvMeta, bool, error)
}
