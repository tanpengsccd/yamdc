package impl

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"yamdc/model"

	"yamdc/searcher/decoder"
	"yamdc/searcher/parser"
	"yamdc/searcher/plugin/api"
	"yamdc/searcher/plugin/constant"
	"yamdc/searcher/plugin/factory"
	putils "yamdc/searcher/utils"
)

type jav321 struct {
	api.DefaultPlugin
}

func (p *jav321) OnMakeHTTPRequest(ctx context.Context, number *model.Number) (*http.Request, error) {
	data := url.Values{}
	data.Set("sn", number.GetNumberID())
	body := data.Encode()
	req, err := http.NewRequest(http.MethodPost, "https://www.jav321.com/search", strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(body)))
	return req, nil
}

func (s *jav321) defaultStringProcessor(v string) string {
	v = strings.Trim(v, ": \t")
	return strings.TrimSpace(v)
}

func (p *jav321) OnDecodeHTTPData(ctx context.Context, data []byte) (*model.AvMeta, bool, error) {
	dec := &decoder.XPathHtmlDecoder{
		NumberExpr:          `//b[contains(text(),"品番")]/following-sibling::node()`,
		TitleExpr:           `/html/body/div[2]/div[1]/div[1]/div[1]/h3/text()`,
		PlotExpr:            `/html/body/div[2]/div[1]/div[1]/div[2]/div[3]/div/text()`,
		ActorListExpr:       `//b[contains(text(),"出演者")]/following-sibling::a[starts-with(@href,"/star")]/text()`,
		ReleaseDateExpr:     `//b[contains(text(),"配信開始日")]/following-sibling::node()`,
		DurationExpr:        `//b[contains(text(),"収録時間")]/following-sibling::node()`,
		StudioExpr:          `//b[contains(text(),"メーカー")]/following-sibling::a[starts-with(@href,"/company")]/text()`,
		LabelExpr:           `//b[contains(text(),"メーカー")]/following-sibling::a[starts-with(@href,"/company")]/text()`,
		SeriesExpr:          `//b[contains(text(),"シリーズ")]/following-sibling::node()`,
		GenreListExpr:       `//b[contains(text(),"ジャンル")]/following-sibling::a[starts-with(@href,"/genre")]/text()`,
		CoverExpr:           `/html/body/div[2]/div[2]/div[1]/p/a/img/@src`,
		PosterExpr:          "",
		SampleImageListExpr: `//div[@class="col-md-3"]/div[@class="col-xs-12 col-md-12"]/p/a/img/@src`,
	}
	rs, err := dec.DecodeHTML(data,
		decoder.WithDefaultStringProcessor(p.defaultStringProcessor),
		decoder.WithReleaseDateParser(parser.DefaultReleaseDateParser(ctx)),
		decoder.WithDurationParser(parser.DefaultDurationParser(ctx)),
	)
	if err != nil {
		return nil, false, err
	}
	putils.EnableDataTranslate(rs)
	return rs, true, nil
}

func init() {
	factory.Register(constant.SSJav321, factory.PluginToCreator(&jav321{}))
}
