package searcher

import (
	"context"
	"yamdc/model"

	"github.com/xxxsen/common/logutil"
	"go.uber.org/zap"
)

type categorySearcher struct {
	defSearcher  []ISearcher
	catSearchers map[model.Category][]ISearcher
}

func NewCategorySearcher(def []ISearcher, cats map[model.Category][]ISearcher) ISearcher {
	return &categorySearcher{defSearcher: def, catSearchers: cats}
}

func (s *categorySearcher) Name() string {
	return "category"
}

func (s *categorySearcher) Search(ctx context.Context, n *model.Number) (*model.AvMeta, bool, error) {
	cat := n.GetCategory()
	//没分类, 那么使用主链进行查询
	//存在分类, 但是分类对应的链没有配置, 则使用主链进行查询
	//如果已经存在分类链, 则不再进行降级
	logger := logutil.GetLogger(ctx).With(zap.String("cat", string(cat)))
	chain := s.defSearcher
	if cat != model.CatDefault {
		if c, ok := s.catSearchers[cat]; ok {
			chain = c
			logger.Debug("use cat chain for search")
		}
	}

	return performGroupSearch(ctx, n, chain)
}
