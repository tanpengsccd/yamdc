package config

import (
	"errors"
	"flag"
	"os"
	"sync"

	"github.com/marcozac/go-jsonc"
	"github.com/xxxsen/common/logger"
)

var (
	globalConfig *Config
	once         sync.Once
)

type CategoryPlugin struct {
	Name    string   `json:"name"`
	Plugins []string `json:"plugins"`
}

type Dependency struct {
	Link    string `json:"link"`
	RelPath string `json:"rel_path"`
}

type Config struct {
	ScanDir string `json:"scan_dir"`
	SaveDir string `json:"save_dir"`
	DataDir string `json:"data_dir"`
	Naming  string `json:"naming"`
	//在提取number前,需要忽略的正则,即匹配到了就会先将其移除后才会去匹配,比如一些广告字段或者域名
	RegexesToReplace [][]string             `json:"regexes_to_replace"`
	PluginConfig     map[string]interface{} `json:"plugin_config"`
	HandlerConfig    map[string]interface{} `json:"handler_config"`
	Plugins          []string               `json:"plugins"`
	CategoryPlugins  []CategoryPlugin       `json:"category_plugins"`
	Handlers         []string               `json:"handlers"`
	ExtraMediaExts   []string               `json:"extra_media_exts"`
	LogConfig        logger.LogConfig       `json:"log_config"`
	Dependencies     []Dependency           `json:"dependencies"`
}

func defaultConfig() *Config {
	return &Config{
		Plugins: []string{
			"javbus",
			"javhoo",
			"airav",
			"javdb",
			"jav321",
			"caribpr",
			"18av",
			"njav",
			"missav",
			"freejavbt",
			"tktube",
			"avsox",
		},
		CategoryPlugins: []CategoryPlugin{
			//如果存在分配配置, 那么当番号被识别为特定分类的场景下, 将会使用分类插件直接查询
			{Name: "FC2", Plugins: []string{"fc2", "18av", "njav", "freejavbt", "tktube", "avsox", "fc2ppvdb"}},
		},
		Handlers: []string{
			"image_transcoder",
			"poster_cropper",
			"watermark_maker",
			"actor_spliter",
			"tag_padder",
			"duration_fixer",
			"number_title",
			"translater",
		},
		LogConfig: logger.LogConfig{
			Level:   "info",
			Console: true,
		},
		Dependencies: []Dependency{
			{Link: "https://github.com/Kagami/go-face-testdata/raw/master/models/shape_predictor_5_face_landmarks.dat", RelPath: "models/shape_predictor_5_face_landmarks.dat"},
			{Link: "https://github.com/Kagami/go-face-testdata/raw/master/models/dlib_face_recognition_resnet_model_v1.dat", RelPath: "models/dlib_face_recognition_resnet_model_v1.dat"},
			{Link: "https://github.com/Kagami/go-face-testdata/raw/master/models/mmod_human_face_detector.dat", RelPath: "models/mmod_human_face_detector.dat"},
			{Link: "https://github.com/esimov/pigo/raw/master/cascade/facefinder", RelPath: "models/facefinder"},
		},
	}
}

func Parse(f string) (*Config, error) {
	raw, err := os.ReadFile(f)
	if err != nil {
		return nil, err
	}
	c := defaultConfig()
	if err := jsonc.Unmarshal(raw, c); err != nil {
		return nil, err
	}
	return c, nil
}

// Init 初始化全局配置
// func init() {

// }

// Shared 获取全局配置实例
func Shared() *Config {
	once.Do(func() {
		var err error
		conf := flag.String("config", "./config.json", "config file")
		flag.Parse()
		globalConfig, err = Parse(*conf)
		if err != nil {
			panic(errors.New("parse config failed, err:" + err.Error()))
		}
	})
	return globalConfig
}
