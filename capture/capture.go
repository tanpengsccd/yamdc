package capture

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"yamdc/debugLogger"
	"yamdc/envflag"
	"yamdc/model"
	"yamdc/nfo"
	"yamdc/number_parser"
	"yamdc/processor"
	"yamdc/store"
	"yamdc/utils"

	"github.com/xxxsen/common/logutil"
	"github.com/xxxsen/common/replacer"
	"github.com/xxxsen/common/trace"
	"go.uber.org/zap"
)

const (
	defaultImageExtName   = ".jpg"
	defaultExtraFanartDir = "extrafanart"
)

var defaultMediaSuffix = []string{".mp4", ".wmv", ".flv", ".mpeg", ".m2ts", ".mts", ".mpe", ".mpg", ".m4v", ".avi", ".mkv", ".rmvb", ".ts", ".mov", ".rm"}

type fcProcessFunc func(ctx context.Context, fc *model.FileContext) error

type Capture struct {
	c      *config
	extMap map[string]struct{}
}

func New(opts ...Option) (*Capture, error) {
	c := &config{}
	for _, opt := range opts {
		opt(c)
	}
	if len(c.SaveDir) == 0 || len(c.ScanDir) == 0 {
		return nil, fmt.Errorf("invalid dir")
	}
	if c.Searcher == nil {
		return nil, fmt.Errorf("no searcher found")
	}
	if c.Processor == nil {
		c.Processor = processor.DefaultProcessor
	}
	if len(c.Naming) == 0 {
		c.Naming = defaultNamingRule
	}
	return &Capture{c: c, extMap: utils.StringListToSet(utils.StringListToLower(append(c.ExtraMediaExtList, defaultMediaSuffix...)))}, nil
}

/* 通过路径文件名 识别电影 信息:电影名,分集数,内嵌中文字幕 等  */
func (c *Capture) resolveFileInfo(fc *model.FileContext, file string) error {
	fc.FileName = filepath.Base(file)
	fc.FileExt = filepath.Ext(file)
	fileNoExt := fc.FileName[:len(fc.FileName)-len(fc.FileExt)]

	// 通过文件名 识别 电影信息 过程
	numberInfo, err := number_parser.Parse(fileNoExt)
	if err != nil {
		return fmt.Errorf("parse number failed, err:%w", err)
	}
	fc.Number = numberInfo
	fc.SaveFileBase = fc.Number.GenerateFileName()
	return nil
}

func (c *Capture) isMediaFile(f string) bool {
	ext := strings.ToLower(filepath.Ext(f))
	if _, ok := c.extMap[ext]; ok {
		return true
	}
	return false
}

/* 读取文件列表,含识别番号的过程，返回一个文件上下文列表。*/
func (c *Capture) readFileList() ([]*model.FileContext, error) {

	fm := utils.NewFileManager()
	fcs := make([]*model.FileContext, 0, 20)
	entries, err := fm.ReadDirSafely(c.c.ScanDir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		fullPath := filepath.Join(c.c.ScanDir, entry.Name())
		// 处理文件名编码
		normalizedPath := fm.NormalizePathForPlatform(fullPath)

		// 检查是否目录
		isDir, err := fm.IsDir(normalizedPath)
		if err != nil {
			return nil, err
		}
		if isDir {
			continue
		}
		// 检查是否媒体文件
		isExist, err := fm.IsExist(normalizedPath)
		if err != nil {
			return nil, err
		}

		if !c.isMediaFile(normalizedPath) {
			continue
		}
		if !isExist {
			// 1. 文件确实不存在了(曾经存在)
			// 2. 文件存在,但是程序无法读取,测试出现过Mac NFS挂载的文件日文能看到,但是无法打开(识别),必须手动重命名后才能打开
			debugLogger.Shared().Error("程序文件无法识别该文件,请检查文件是否存在,如存在请手动命名再试", zap.String("file", normalizedPath))
			continue
		}
		fc := &model.FileContext{FullFilePath: normalizedPath}
		// 通过路径文件名 识别电影 信息
		if err := c.resolveFileInfo(fc, normalizedPath); err != nil {
			return nil, err
		}
		fcs = append(fcs, fc)
	}
	if len(fcs) == 0 {
		return nil, fmt.Errorf("no valid files found in directory: %s", c.c.ScanDir)

	}

	return fcs, nil
}

// Run 执行捕获过程，主要负责读取文件列表、展示数字信息和处理文件列表。
// 该函数接收一个 context.Context 类型的参数 ctx，用于控制操作的取消或超时。
func (c *Capture) Run(ctx context.Context) error {
	// 读取文件列表，如果读取失败，则返回错误信息。
	fcs, err := c.readFileList()
	if err != nil {
		return fmt.Errorf("read file list failed, err:%w", err)
	}
	debugLogger.Shared().Sugar().Debugf("start read local file!⬇️⬇️⬇️⬇️⬇️⬇️⬇️⬇️⬇️⬇️⬇️⬇️⬇️⬇️⬇️")
	// 扫描本地文件路径视频以获取文件基础信息
	c.displayNumberInfo(ctx, fcs)
	debugLogger.Shared().Sugar().Debugf("finish read local file success!⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️")
	// TODO 处理文件列表
	debugLogger.Shared().Sugar().Debugf("start process file!⬇️⬇️⬇️⬇️⬇️⬇️⬇️⬇️⬇️⬇️⬇️⬇️⬇️⬇️⬇️")

	if err := c.processFileList(ctx, fcs); err != nil {
		debugLogger.Shared().Sugar().Debugf("failed process file !⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️")
		return fmt.Errorf("proc file list failed, err:%w", err)
	} else {
		debugLogger.Shared().Sugar().Debugf("finish process file !⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️⬆️")
	}

	return nil
}

// CondString 当 value 非空时返回 zap.String，否则返回 zap.Skip()
func CondString(key, value string) zap.Field {
	if value == "" {
		return zap.Skip()
	} else {
		return zap.String(key, value)
	}

}
func CondBool(key string, value bool, trueString string) zap.Field {
	if !value {
		return zap.Skip()
	} else {
		return zap.String(key, trueString)
	}

}
func (c *Capture) displayNumberInfo(ctx context.Context, fcs []*model.FileContext) {
	logutil.GetLogger(ctx).Info("read local movie file success! 💼--------------------------------------------------", zap.Int("count", len(fcs)))
	for _, item := range fcs {
		// 打印文件信息, 只打印item.Number 为true的信息,或者有的信息
		logutil.GetLogger(ctx).Info("file info",
			CondString("number", item.Number.NumberId),
			CondString("ep", item.Number.Episode),
			CondBool("cnsub", item.Number.IsCnSub, "🀄️"),
			CondBool("uncensored", item.Number.IsUncensored, "🔳"),
			CondBool("4k", item.Number.Is4k, "📺"),
			CondBool("cracked", item.Number.IsCracked, "🔓"),
			CondBool("leaked", item.Number.IsLeaked, "💧"),
			CondString("cat", item.Number.Cat.String()),
			zap.String("file", item.FileName),
		)

	}
}

// 刮削提取的文件信息
func (c *Capture) processFileList(ctx context.Context, fcs []*model.FileContext) error {
	var outErr error
	for _, item := range fcs {
		if err := c.processOneFile(ctx, item); err != nil {
			outErr = err
			logutil.GetLogger(ctx).Error("process file failed", zap.Error(err), zap.String("file", item.FullFilePath))
			continue
		}
		logutil.GetLogger(ctx).Info("process file succ", zap.String("file", item.FullFilePath))
	}
	return outErr
}

func (c *Capture) resolveSaveDir(fc *model.FileContext) error {
	ts := time.UnixMilli(fc.Meta.ReleaseDate)
	date := ts.Format(time.DateOnly)
	year := fmt.Sprintf("%d", ts.Year())
	month := fmt.Sprintf("%d", ts.Month())
	actor := "佚名"
	if len(fc.Meta.Actors) > 0 {
		actor = utils.BuildAuthorsName(fc.Meta.Actors, 256)
	}
	m := map[string]interface{}{
		NamingReleaseDate:  date,
		NamingReleaseYear:  year,
		NamingReleaseMonth: month,
		NamingActor:        actor,
		NamingNumber:       fc.Number.GetNumberID(),
	}
	naming := replacer.ReplaceByMap(c.c.Naming, m)
	if len(naming) == 0 {
		return fmt.Errorf("invalid naming")
	}
	fc.SaveDir = filepath.Join(c.c.SaveDir, naming)
	return nil
}

func (c *Capture) doSearch(ctx context.Context, fc *model.FileContext) error {
	meta, ok, err := c.c.Searcher.Search(ctx, fc.Number)
	if err != nil {
		return fmt.Errorf("search number failed, number:%s, err:%w", fc.Number.GetNumberID(), err)
	}
	if !ok {
		return fmt.Errorf("search item not found")
	}
	if meta.Number != fc.Number.GetNumberID() {
		logutil.GetLogger(ctx).Warn("number not match, may be re-generated, ignore", zap.String("search", meta.Number), zap.String("file", fc.Number.GetNumberID()))
	}
	fc.Meta = meta
	return nil
}

func (c *Capture) doProcess(ctx context.Context, fc *model.FileContext) error {
	//执行处理流程, 用于补齐数据或者数据转换
	if err := c.c.Processor.Process(ctx, fc); err != nil {
		//process 不作为关键路径, 一个meta能否可用取决于后续的verify逻辑
		logutil.GetLogger(ctx).Error("process meta failed, go next", zap.Error(err))
	}
	return nil
}

func (c *Capture) doNaming(ctx context.Context, fc *model.FileContext) error {
	//构建保存目录地址
	if err := c.resolveSaveDir(fc); err != nil {
		return fmt.Errorf("resolve save dir failed, err:%w", err)
	}
	//创建必要的目录
	if err := os.MkdirAll(fc.SaveDir, 0755); err != nil {
		return fmt.Errorf("make save dir failed, err:%w", err)
	}
	if err := os.MkdirAll(filepath.Join(fc.SaveDir, defaultExtraFanartDir), 0755); err != nil {
		return fmt.Errorf("make fanart dir failed, err:%w", err)
	}
	//数据重命名
	if err := c.renameMetaField(fc); err != nil {
		return fmt.Errorf("rename meta field failed, err:%w", err)
	}
	return nil
}

func (c *Capture) doSaveData(ctx context.Context, fc *model.FileContext) error {
	//保存元数据并将影片移入指定目录
	if err := c.saveMediaData(ctx, fc); err != nil {
		return fmt.Errorf("save meta data failed, err:%w", err)
	}
	return nil
}

func (c *Capture) doExport(ctx context.Context, fc *model.FileContext) error {
	// 导出jellyfin需要的nfo信息
	if err := c.exportNFOData(fc); err != nil {
		return fmt.Errorf("export nfo data failed, err:%w", err)
	}
	return nil
}

func (c *Capture) doMetaVerify(ctx context.Context, fc *model.FileContext) error {
	//全部处理完后必须要保证当前的元数据至少有title, number, cover, title
	if len(fc.Meta.Title) == 0 {
		return fmt.Errorf("no title")
	}
	if len(fc.Meta.Number) == 0 {
		return fmt.Errorf("no number found")
	}
	if fc.Meta.Cover == nil || len(fc.Meta.Cover.Name) == 0 || len(fc.Meta.Cover.Key) == 0 {
		return fmt.Errorf("invalid cover")
	}
	if fc.Meta.Poster == nil || len(fc.Meta.Poster.Name) == 0 || len(fc.Meta.Poster.Key) == 0 {
		return fmt.Errorf("invalid poster")
	}
	return nil
}

func (c *Capture) processOneFile(ctx context.Context, fc *model.FileContext) error {
	ctx = trace.WithTraceId(ctx, "TID:N:"+fc.Number.GetNumberID())
	steps := []struct {
		name string
		fn   fcProcessFunc
	}{
		{"search", c.doSearch},
		{"process", c.doProcess},
		{"metaverify", c.doMetaVerify},
		{"naming", c.doNaming},
		{"savedata", c.doSaveData},
		{"nfo", c.doExport},
	}
	logger := logutil.GetLogger(ctx).With(zap.String("file", fc.FileName))
	for idx, step := range steps {
		log := logger.With(zap.Int("idx", idx), zap.String("name", step.name))
		log.Debug("step start")
		if err := step.fn(ctx, fc); err != nil {
			log.Error("proc step failed", zap.Error(err))
			return err
		}
		log.Debug("step end")
	}
	logger.Info("process succ",
		zap.String("number_id", fc.Number.GetNumberID()),
		zap.String("scrape_source", fc.Meta.ExtInfo.ScrapeInfo.Source),
		zap.String("release_date", utils.FormatTimeToDate(fc.Meta.ReleaseDate)),
		zap.Int("duration", int(fc.Meta.Duration)),
		zap.Int("sample_img_cnt", len(fc.Meta.SampleImages)),
		zap.Strings("genres", fc.Meta.Genres),
		zap.Strings("actors", fc.Meta.Actors),
		zap.String("title", fc.Meta.Title),
		zap.String("plot", fc.Meta.Plot),
	)
	return nil
}

func (c *Capture) renameMetaField(fc *model.FileContext) error {
	if fc.Meta.Cover != nil {
		fc.Meta.Cover.Name = fmt.Sprintf("%s-fanart%s", fc.SaveFileBase, defaultImageExtName)
	}
	if fc.Meta.Poster != nil {
		fc.Meta.Poster.Name = fmt.Sprintf("%s-poster%s", fc.SaveFileBase, defaultImageExtName)
	}
	for idx, item := range fc.Meta.SampleImages { //TODO:这里需要构建子目录, 看看有没有更好的做法
		item.Name = fmt.Sprintf("%s/%s-sample-%d%s", defaultExtraFanartDir, fc.SaveFileBase, idx, defaultImageExtName)
	}
	return nil
}

func (c *Capture) saveMediaData(ctx context.Context, fc *model.FileContext) error {
	images := make([]*model.File, 0, len(fc.Meta.SampleImages)+2)
	if fc.Meta.Cover != nil {
		images = append(images, fc.Meta.Cover)
	}
	if fc.Meta.Poster != nil {
		images = append(images, fc.Meta.Poster)
	}
	images = append(images, fc.Meta.SampleImages...)
	for _, image := range images {
		target := filepath.Join(fc.SaveDir, image.Name)
		logger := logutil.GetLogger(context.Background()).With(zap.String("image", image.Name), zap.String("key", image.Key), zap.String("target", target))

		data, err := store.GetData(ctx, image.Key)
		if err != nil {
			logger.Error("read image data failed", zap.Error(err))
			return err
		}

		if err := os.WriteFile(target, data, 0644); err != nil {
			logger.Error("write image failed", zap.Error(err))
			return err
		}
		logger.Debug("write image succ")
	}
	movie := filepath.Join(fc.SaveDir, fc.SaveFileBase+fc.FileExt)
	if err := c.moveMovie(fc, fc.FullFilePath, movie); err != nil {
		return fmt.Errorf("move movie to dst dir failed, err:%w", err)
	}
	return nil
}

func (c *Capture) moveMovie(fc *model.FileContext, src string, dst string) error {
	// 暂时不移动, 打印 移动
	//TODO: 暂时不移动, 打印 移动
	// debugLogger.Shared().Debugw("🐛:move movie to dst dir", src, dst)
	// return nil
	if envflag.IsEnableLinkMode() {
		return c.moveMovieByLink(fc, src, dst)
	}
	return c.moveMovieDirect(fc, src, dst)
}

func (c *Capture) moveMovieByLink(_ *model.FileContext, src, dst string) error {
	err := os.Symlink(src, dst)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil
		}
	}
	return err
}

func (c *Capture) moveMovieDirect(_ *model.FileContext, src, dst string) error {
	return utils.NewFileManager().Move(src, dst)
}

func (c *Capture) exportNFOData(fc *model.FileContext) error {
	mov, err := utils.ConvertMetaToMovieNFO(fc.Meta)
	if err != nil {
		return fmt.Errorf("convert meta to movie nfo failed, err:%w", err)
	}
	save := filepath.Join(fc.SaveDir, fc.SaveFileBase+".nfo")
	if err := nfo.WriteMovieToFile(save, mov); err != nil {
		return fmt.Errorf("write movie nfo failed, err:%w", err)
	}
	return nil
}
