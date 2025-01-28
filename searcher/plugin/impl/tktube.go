package impl

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"yamdc/model"

	"yamdc/searcher/decoder"
	"yamdc/searcher/parser"
	"yamdc/searcher/plugin/api"
	"yamdc/searcher/plugin/constant"
	"yamdc/searcher/plugin/factory"
	"yamdc/searcher/plugin/meta"
	"yamdc/searcher/plugin/twostep"
)

type tktube struct {
	api.DefaultPlugin
}

func (p *tktube) OnMakeHTTPRequest(ctx context.Context, n *model.Number) (*http.Request, error) {
	nid := strings.ReplaceAll(n.GetNumberID(), "-", "--")
	uri := fmt.Sprintf("https://tktube.com/zh/search/%s/", nid)
	return http.NewRequest(http.MethodGet, uri, nil)
}

func (p *tktube) OnHandleHTTPRequest(ctx context.Context, invoker api.HTTPInvoker, req *http.Request) (*http.Response, error) {
	numberId := strings.ToUpper(meta.GetNumberId(ctx))
	return twostep.HandleXPathTwoStepSearch(ctx, invoker, req, &twostep.XPathTwoStepContext{
		Ps: []*twostep.XPathPair{
			{
				Name:  "links",
				XPath: `//div[@id="list_videos_videos_list_search_result_items"]/div/a/@href`,
			},
			{
				Name:  "names",
				XPath: `//div[@id="list_videos_videos_list_search_result_items"]/div/a/strong[@class="title"]/text()`,
			},
		},
		LinkSelector: func(ps []*twostep.XPathPair) (string, bool, error) {
			links := ps[0].Result
			names := ps[1].Result
			for i := 0; i < len(links); i++ {
				if strings.Contains(strings.ToUpper(names[i]), numberId) {
					return links[i], true, nil
				}
			}
			return "", false, nil
		},
		ValidStatusCode:       []int{http.StatusOK},
		CheckResultCountMatch: true,
		LinkPrefix:            "",
	})
}

func (p *tktube) OnDecodeHTTPData(ctx context.Context, data []byte) (*model.AvMeta, bool, error) {
	dec := decoder.XPathHtmlDecoder{
		TitleExpr:           `//div[@class="headline"]/h1/text()`,
		PlotExpr:            "",
		ActorListExpr:       `//div[contains(text(), "女優:")]/a[contains(@href, "models")]/text()`,
		ReleaseDateExpr:     `//div[@class="item"]/span[contains(text(), "加入日期:")]/em/text()`,
		DurationExpr:        `//div[@class="item"]/span[contains(text(), "時長:")]/em/text()`,
		StudioExpr:          "",
		LabelExpr:           "",
		DirectorExpr:        "",
		SeriesExpr:          "",
		GenreListExpr:       `//div[contains(text(), "標籤:")]/a[contains(@href, "tags")]/text()`,
		CoverExpr:           `//meta[@property="og:image"]/@content`,
		PosterExpr:          "",
		SampleImageListExpr: "",
	}
	res, err := dec.DecodeHTML(data,
		decoder.WithDurationParser(parser.DefaultHHMMSSDurationParser(ctx)),
		decoder.WithReleaseDateParser(parser.DefaultReleaseDateParser(ctx)),
	)
	if err != nil {
		return nil, false, err
	}
	res.Number = meta.GetNumberId(ctx)
	return res, true, nil
}

func init() {
	factory.Register(constant.SSTKTube, factory.PluginToCreator(&tktube{}))
}
