package searcher

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
	"yamdc/client"
	"yamdc/envflag"
	"yamdc/hasher"
	"yamdc/model"
	"yamdc/searcher/plugin/api"
	"yamdc/searcher/plugin/meta"
	"yamdc/store"
	"yamdc/useragent"

	"github.com/xxxsen/common/logutil"
	"go.uber.org/zap"
)

const (
	defaultPageSearchCacheExpire = 30 * 24 * time.Hour
)

type DefaultSearcher struct {
	name    string
	ua      string
	invoker api.HTTPInvoker
	plg     api.IPlugin
}

func MustNewDefaultSearcher(name string, plg api.IPlugin) ISearcher {
	s, err := NewDefaultSearcher(name, plg)
	if err != nil {
		panic(err)
	}
	return s
}

func defaultInvoker() api.HTTPInvoker {
	basicClient := client.NewClient()
	return func(ctx context.Context, req *http.Request) (*http.Response, error) {
		return basicClient.Do(req)
	}
}

func NewDefaultSearcher(name string, plg api.IPlugin) (ISearcher, error) {
	invoker := plg.OnHTTPClientInit()
	if invoker == nil {
		invoker = defaultInvoker()
	}
	ss := &DefaultSearcher{
		name:    name,
		invoker: invoker,
		plg:     plg,
		ua:      useragent.Select(),
	}
	return ss, nil
}

func (p *DefaultSearcher) Name() string {
	return p.name
}

func (p *DefaultSearcher) setDefaultHttpOptions(req *http.Request) error {
	if len(req.UserAgent()) == 0 {
		req.Header.Set("User-Agent", p.ua)
	}
	if len(req.Referer()) == 0 {
		req.Header.Set("Referer", fmt.Sprintf("%s://%s/", req.URL.Scheme, req.URL.Host))
	}
	return nil
}

func (p *DefaultSearcher) decorateRequest(ctx context.Context, req *http.Request) error {
	if err := p.plg.OnDecorateRequest(ctx, req); err != nil {
		return err
	}
	if err := p.setDefaultHttpOptions(req); err != nil {
		return err
	}
	return nil
}

func (p *DefaultSearcher) decorateImageRequest(ctx context.Context, req *http.Request) error {
	if err := p.plg.OnDecorateMediaRequest(ctx, req); err != nil {
		return err
	}
	if err := p.setDefaultHttpOptions(req); err != nil {
		return err
	}
	return nil
}

func (p *DefaultSearcher) invokeHTTPRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	if err := p.decorateRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("decorate request failed, err:%w", err)
	}
	return p.invoker(ctx, req)
}

func (p *DefaultSearcher) onRetriveData(ctx context.Context, req *http.Request, number *model.Number) ([]byte, error) {
	key := p.name + ":" + number.GetNumberID()
	dataLoader := func() ([]byte, error) {
		rsp, err := p.plg.OnHandleHTTPRequest(ctx, p.invokeHTTPRequest, req)
		if err != nil {
			return nil, fmt.Errorf("do request failed, err:%w", err)
		}
		isSearchSucc, err := p.plg.OnPrecheckResponse(ctx, req, rsp)
		if err != nil {
			return nil, fmt.Errorf("precheck responnse failed, err:%w", err)
		}
		if !isSearchSucc {
			return nil, fmt.Errorf("no data found")
		}
		if rsp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("invalid http status code:%d", rsp.StatusCode)
		}
		defer rsp.Body.Close()
		data, err := client.ReadHTTPData(rsp)
		if err != nil {
			return nil, fmt.Errorf("read body failed, err:%w", err)
		}
		return data, nil
	}
	if !envflag.IsEnableSearchMetaCache() {
		return dataLoader()
	}
	return store.LoadData(ctx, key, defaultPageSearchCacheExpire, dataLoader)
}

func (p *DefaultSearcher) Search(ctx context.Context, number *model.Number) (*model.AvMeta, bool, error) {
	ctx = meta.SetNumberId(ctx, number.GetNumberID())
	ok, err := p.plg.OnPrecheckRequest(ctx, number)
	if err != nil {
		return nil, false, fmt.Errorf("precheck failed, err:%w", err)
	}
	if !ok {
		return nil, false, nil
	}
	req, err := p.plg.OnMakeHTTPRequest(ctx, number)
	if err != nil {
		return nil, false, fmt.Errorf("make http request failed, err:%w", err)
	}
	data, err := p.onRetriveData(ctx, req, number)
	if err != nil {
		return nil, false, err
	}
	meta, decodeSucc, err := p.plg.OnDecodeHTTPData(ctx, data)
	if err != nil {
		return nil, false, fmt.Errorf("decode http data failed, err:%w", err)
	}
	if !decodeSucc {
		return nil, false, nil
	}
	//重建不规范的元数据
	p.fixMeta(req, meta)
	//将远程数据保存到本地, 并替换文件key
	p.storeImageData(ctx, meta)
	if err := p.verifyMeta(meta); err != nil {
		logutil.GetLogger(ctx).Error("verify meta not pass, treat as not found", zap.Error(err), zap.String("plugin", p.name))
		return nil, false, nil
	}
	meta.ExtInfo.ScrapeInfo.Source = p.name
	meta.ExtInfo.ScrapeInfo.DateTs = time.Now().UnixMilli()
	return meta, true, nil
}

func (p *DefaultSearcher) verifyMeta(meta *model.AvMeta) error {
	if meta.Cover == nil || len(meta.Cover.Name) == 0 {
		return fmt.Errorf("no cover")
	}
	if len(meta.Number) == 0 {
		return fmt.Errorf("no number")
	}
	if len(meta.Title) == 0 {
		return fmt.Errorf("no title")
	}
	if meta.ReleaseDate == 0 {
		return fmt.Errorf("no release_date")
	}
	return nil
}

func (p *DefaultSearcher) fixMeta(req *http.Request, meta *model.AvMeta) {
	meta.Number = strings.ToUpper(meta.Number)
	prefix := req.URL.Scheme + "://" + req.URL.Host
	if meta.Cover != nil {
		p.fixSingleURL(req, &meta.Cover.Name, prefix)
	}
	if meta.Poster != nil {
		p.fixSingleURL(req, &meta.Poster.Name, prefix)
	}
	for i := 0; i < len(meta.SampleImages); i++ {
		p.fixSingleURL(req, &meta.SampleImages[i].Name, prefix)
	}
}

func (p *DefaultSearcher) fixSingleURL(req *http.Request, input *string, prefix string) {
	if strings.HasPrefix(*input, "//") {
		*input = req.URL.Scheme + ":" + *input
		return
	}
	if strings.HasPrefix(*input, "/") {
		*input = prefix + *input
		return
	}
}

func (p *DefaultSearcher) storeImageData(ctx context.Context, in *model.AvMeta) {
	images := make([]string, 0, len(in.SampleImages)+2)
	if in.Cover != nil {
		images = append(images, in.Cover.Name)
	}
	if in.Poster != nil {
		images = append(images, in.Poster.Name)
	}
	for _, item := range in.SampleImages {
		images = append(images, item.Name)
	}
	imageDataMap := p.saveRemoteURLData(ctx, images)
	if in.Cover != nil {
		in.Cover.Key = imageDataMap[in.Cover.Name]
		//如果没有成功下载到数据, 那么直接置空
		if len(in.Cover.Key) == 0 {
			in.Cover = nil
		}
	}
	if in.Poster != nil {
		in.Poster.Key = imageDataMap[in.Poster.Name]
		if len(in.Poster.Key) == 0 {
			in.Poster = nil
		}
	}
	rebuildSampleList := make([]*model.File, 0, len(in.SampleImages))
	for _, item := range in.SampleImages {
		item.Key = imageDataMap[item.Name]
		rebuildSampleList = append(rebuildSampleList, item)
	}
	in.SampleImages = rebuildSampleList
}

func (p *DefaultSearcher) saveRemoteURLData(ctx context.Context, urls []string) map[string]string {
	rs := make(map[string]string, len(urls))
	for _, url := range urls {
		if len(url) == 0 {
			continue
		}
		logger := logutil.GetLogger(context.Background()).With(zap.String("url", url))
		key := hasher.ToSha1(url)
		if ok, _ := store.IsDataExist(ctx, key); ok {
			rs[url] = key
			continue
		}
		data, err := p.fetchImageData(ctx, url)
		if err != nil {
			logger.Error("fetch image data failed", zap.Error(err))
			continue
		}
		err = store.PutData(ctx, key, data)
		if err != nil {
			logger.Error("put image data to store failed", zap.Error(err))
		}
		rs[url] = key
	}
	return rs
}

func (p *DefaultSearcher) fetchImageData(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("make request for url:%s failed, err:%w", url, err)
	}
	if err := p.decorateImageRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("decode request failed, err:%w", err)
	}
	rsp, err := p.invoker(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("get url data failed, err:%w", err)
	}

	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get url data http code not ok, code:%d", rsp.StatusCode)
	}
	data, err := client.ReadHTTPData(rsp)
	if err != nil {
		return nil, fmt.Errorf("read url data failed, err:%w", err)
	}
	return data, nil
}
