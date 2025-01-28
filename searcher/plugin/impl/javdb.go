package impl

import (
	"context"
	"fmt"
	"net/http"
	"yamdc/model"

	"yamdc/number_parser"

	"yamdc/searcher/decoder"
	"yamdc/searcher/parser"
	"yamdc/searcher/plugin/api"
	"yamdc/searcher/plugin/constant"
	"yamdc/searcher/plugin/factory"
	"yamdc/searcher/plugin/meta"
	"yamdc/searcher/plugin/twostep"
	"yamdc/searcher/utils"
)

type javdb struct {
	api.DefaultPlugin
}

func (p *javdb) OnMakeHTTPRequest(ctx context.Context, number *model.Number) (*http.Request, error) {
	link := fmt.Sprintf("https://javdb.com/search?q=%s&f=all", number.GetNumberID())
	return http.NewRequest(http.MethodGet, link, nil)
}

func (p *javdb) OnHandleHTTPRequest(ctx context.Context, invoker api.HTTPInvoker, req *http.Request) (*http.Response, error) {
	return twostep.HandleXPathTwoStepSearch(ctx, invoker, req, &twostep.XPathTwoStepContext{
		Ps: []*twostep.XPathPair{
			{
				Name:  "read-link",
				XPath: `//div[@class="movie-list h cols-4 vcols-8"]/div[@class="item"]/a/@href`,
			},
			{
				Name:  "read-number",
				XPath: `//div[@class="movie-list h cols-4 vcols-8"]/div[@class="item"]/a/div[@class="video-title"]/strong`,
			},
		},
		LinkSelector: func(ps []*twostep.XPathPair) (string, bool, error) {
			linklist := ps[0].Result
			numberlist := ps[1].Result
			num := number_parser.GetCleanID(meta.GetNumberId(ctx))
			for idx, numberItem := range numberlist {
				link := linklist[idx]
				if number_parser.GetCleanID(numberItem) == num {
					return link, true, nil
				}
			}
			return "", false, nil
		},
		ValidStatusCode:       []int{http.StatusOK},
		CheckResultCountMatch: true,
		LinkPrefix:            "https://javdb.com",
	})
}

func (p *javdb) OnDecodeHTTPData(ctx context.Context, data []byte) (*model.AvMeta, bool, error) {
	dec := decoder.XPathHtmlDecoder{
		NumberExpr:          `//a[@class="button is-white copy-to-clipboard"]/@data-clipboard-text`,
		TitleExpr:           `//h2[@class="title is-4"]/strong[@class="current-title"]`,
		PlotExpr:            "",
		ActorListExpr:       `//div[strong[contains(text(), "演員")]]/span[@class="value"]/a`,
		ReleaseDateExpr:     `//div[strong[contains(text(), "日期")]]/span[@class="value"]`,
		DurationExpr:        `//div[strong[contains(text(), "時長")]]/span[@class="value"]`,
		StudioExpr:          `//div[strong[contains(text(), "片商")]]/span[@class="value"]`,
		LabelExpr:           "",
		DirectorExpr:        "",
		SeriesExpr:          `//div[strong[contains(text(), "系列")]]/span[@class="value"]`,
		GenreListExpr:       `//div[strong[contains(text(), "類別")]]/span[@class="value"]/a`,
		CoverExpr:           `//div[@class="column column-video-cover"]/a/img/@src`,
		PosterExpr:          "",
		SampleImageListExpr: `//div[@class="tile-images preview-images"]/a[@class="tile-item"]/@href`,
	}
	meta, err := dec.DecodeHTML(data,
		decoder.WithReleaseDateParser(parser.DefaultReleaseDateParser(ctx)),
		decoder.WithDurationParser(parser.DefaultDurationParser(ctx)),
	)
	if err != nil {
		return nil, false, err
	}
	if len(meta.Number) == 0 {
		return nil, false, nil
	}
	utils.EnableDataTranslate(meta)
	return meta, true, nil
}

func init() {
	factory.Register(constant.SSJavDB, factory.PluginToCreator(&javdb{}))
}
