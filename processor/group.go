package processor

import (
	"av-capture/model"
	"context"

	"github.com/xxxsen/common/logutil"
	"go.uber.org/zap"
)

type group struct {
	ps []IProcessor
}

func NewGroup(ps []IProcessor) IProcessor {
	return &group{ps: ps}
}

func (g *group) Name() string {
	return "group"
}

func (g *group) Process(ctx context.Context, meta *model.AvMeta) error {
	for _, p := range g.ps {
		err := p.Process(ctx, meta)
		if err == nil {
			continue
		}
		logutil.GetLogger(ctx).Error("process failed", zap.Error(err), zap.String("name", p.Name()), zap.Bool("optional", p.IsOptional()))
		if !p.IsOptional() {
			return err
		}
	}
	return nil
}

func (g *group) IsOptional() bool {
	return false
}
