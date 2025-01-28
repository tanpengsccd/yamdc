package number_parser

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"yamdc/config"
	"yamdc/debugLogger"
	"yamdc/model"
	"yamdc/utils"

	"github.com/dlclark/regexp2"
	"github.com/samber/lo"
)

type Number = model.Number
type suffixInfoResolveFunc func(info *Number, normalizedSuffix string) bool
type numberInfoResolveFunc func(info *Number, number string)

var defaultSuffixResolverList = []suffixInfoResolveFunc{
	resolveIsChineseSubTitle,
	resolveCDInfo,
	resolve4K,
	resolveLeak,
}

var defaultNumberInfoResolverList = []numberInfoResolveFunc{
	resolveIsUncensorMovie,
}

func extractSuffix(str string) (string, bool) {
	for i := len(str) - 1; i >= 0; i-- {
		if str[i] == '_' || str[i] == '-' {
			return str[i:], true
		}
	}
	return "", false
}

func tryResolveSuffix(info *Number, suffix string) bool {
	normalizedSuffix := strings.ToUpper(suffix[1:])
	for _, resolver := range defaultSuffixResolverList {
		if resolver(info, normalizedSuffix) {
			return true
		}
	}
	return false
}

func resolveSuffixInfo(info *Number, str string) string {
	for {
		suffix, ok := extractSuffix(str)
		if !ok {
			return str
		}
		if !tryResolveSuffix(info, suffix) {
			return str
		}
		str = str[:len(str)-len(suffix)]
	}
}

func resolveCDInfo(info *Number, str string) bool {
	if !strings.HasPrefix(str, model.DefaultSuffixMultiCD) {
		return false
	}
	strNum := str[2:]
	// num, err := strconv.ParseInt(strNum, 10, 64)
	// if err != nil {
	// 	return false
	// }
	// info.isMultiCD = true
	info.Episode = strNum

	return true
}

func resolveLeak(info *Number, str string) bool {
	if str != model.DefaultSuffixLeak {
		return false
	}
	info.IsLeaked = true
	return true
}

func resolve4K(info *Number, str string) bool {
	if str != model.DefaultSuffix4K {
		return false
	}
	info.Is4k = true
	return true
}

func resolveIsChineseSubTitle(info *Number, str string) bool {
	if str != model.DefaultSuffixChineseSubtitle {
		return false
	}
	info.IsCnSub = true
	return true
}

func resolveNumberInfo(info *Number, number string) {
	for _, resolver := range defaultNumberInfoResolverList {
		resolver(info, number)
	}
}

func resolveIsUncensorMovie(info *Number, str string) {
	if IsUncensorMovie(str) {
		info.IsUncensored = true
	}
}

func ParseWithFileName(f string) (*Number, error) {
	filename := filepath.Base(f)
	fileext := filepath.Ext(f)
	filenoext := filename[:len(filename)-len(fileext)]
	return Parse(filenoext)
}

func Parse(str string) (*Number, error) {
	if len(str) == 0 {
		return nil, fmt.Errorf("empty number str")
	}
	// 1.清楚文件名中的干扰字段
	fileNameCleaned := cleanFileName(str)
	//TODO  抽取清楚度 字段,暂时只做了抽取4K相关字段
	is4k, fileNameWithoutResolution := extractIs4k(fileNameCleaned)
	// 2 ❤️ 获取 番号信息
	codeInfo := func() *CodeInfo {

		// 抽取文件路径中可能存在的尾部集数，和抽取尾部集数的后的文件路径
		episode_suffix, fileNameExtractSuffixEpisode := extractSuffixEpisode(fileNameWithoutResolution)
		if episode_suffix != "" {

			// 如果使用先 抽取文件路径中可能存在的尾部集数，和抽取尾部集数的后的文件路径  获取到番号，则返回
			codeInfoTemp := extractCode(fileNameExtractSuffixEpisode)
			if codeInfoTemp.Code != "" {
				// 如果 episode_suffix和codeInfoTemp.Episode都存在,且其中有“C” ,则必定其中一个表示 IsCnSubs ,另一个表示Eposide
				if episode_suffix != "" && codeInfoTemp.Episode != "" && lo.Contains([]string{codeInfoTemp.Episode, episode_suffix}, "C") {
					codeInfoTemp.IsCnSubs = true
					if codeInfoTemp.Episode == "C" {
						codeInfoTemp.Episode = episode_suffix
					}
				} else {
					codeInfoTemp.Episode = lo.CoalesceOrEmpty(codeInfoTemp.Episode, episode_suffix)
				}
				return codeInfoTemp
			} else {
				//  如果番号中没有提取到,则可能是 中提取尾部episode_suffix本身是番号一部分 比如 "052624-01"中“01”, 则尝试重新用上一级的name提取
				codeInfoTemp2 := extractCode(fileNameWithoutResolution)
				return codeInfoTemp2
			}
		} else {
			// 如果没有提取到尾部集数,则直接提取番号
			return extractCode(fileNameWithoutResolution)
		}

	}()
	// ------
	// if strings.Contains(str, ".") {
	// 	return nil, fmt.Errorf("should not contain extname, str:%s", str)
	// }
	codeInfo.Code = strings.ToUpper(codeInfo.Code)
	rs := &Number{
		NumberId:     "",
		IsCnSub:      false,
		IsUncensored: false,
		Is4k:         false,
	}
	// //部分番号需要进行改写
	// number = rewriteNumber(number)
	// //提取后缀信息并对番号进行裁剪
	// number = resolveSuffixInfo(rs, number)
	// //通过番号直接填充信息(不进行裁剪)
	// resolveNumberInfo(rs, number)
	// rs.numberId = number
	rs.NumberId = codeInfo.Code
	rs.Episode = codeInfo.Episode
	rs.IsCnSub = codeInfo.IsCnSubs
	rs.IsLeaked = codeInfo.IsLeaked
	rs.IsUncensored = codeInfo.IsUncensored
	rs.Is4k = is4k
	//暂时只处理数字集数的情况
	// debugLogger.Shared().Debugw("ParseWithFileName",
	// 	"number", rs.NumberId,
	// 	"isChineseSubtitle", rs.IsCnSub,

	// 	"episode_suffix", episode_suffix,
	// 	"fileNameExtractSuffixEpisode", fileNameExtractSuffixEpisode,
	// 	"isLeak", rs.IsLeaked,
	// 	"is4k", rs.Is4k,
	// 	"cat", rs.Cat,
	// )

	rs.Cat = model.DetermineCategory(rs.NumberId)
	return rs, nil
}

func ReorganizeAllNumbers(fcs []*model.FileContext) {
	// 分析文件列表, 重新调整: 识别到分集“C” 也可能是 中文字幕,需要通过判断同集目录是否含有B集来判断。
	// 1. 遍历 fcs, 找到所有的番号含C的
	for _, fc := range fcs {
		// 如果是单集, 则跳过
		if fc.Number.Episode == "C" {
			// 查找出 fcs中 同GetNumberID 的所有文件
			sameNumberIDs := lo.Filter(fcs, func(itemIn *model.FileContext, _ int) bool {
				return itemIn.Number.GetNumberID() == fc.Number.GetNumberID()
			})
			// 找出是否有同id中 含有 B 集且目录相同的
			isSameDirContainB := lo.ContainsBy(sameNumberIDs, func(itemIn *model.FileContext) bool {
				return (itemIn.Number.Episode == "B") && itemIn.Dir(0) == fc.Dir(0)
			})
			if isSameDirContainB {
				fc.Number.Episode = "C"
				fc.Number.SetIsChineseSubtitle(false)
			} else {
			}

		} else {
		}
	}
}

/* 清理文件名中的干扰字段 */
func cleanFileName(str string) string {
	// 读取配置RegexesToReplace,用来替换或者移除无关字段
	str = func(str string) string {
		cfg := config.Shared()
		newStr := str
		for _, regex_to_replace := range cfg.RegexesToReplace {
			// 使用三方regex2 才能 前瞻/后顾断言
			re, err := regexp2.Compile(regex_to_replace[0], regexp2.RE2)
			if err != nil {
				fmt.Println("正则错误:", err)
			}
			repl := regex_to_replace[1]
			// 打印替换后的字符串

			newStr, err = re.Replace(newStr, repl, -1, -1)
			if err != nil {
				fmt.Println("替换错误:", err)
			}
		}
		return newStr
	}(str)

	return str
}

// 读取清晰度: 暂时只读取是否4k
func extractIs4k(originName string) (bool, string) {
	// 通过抽取 u(ltra).+hd  2160p   4k 忽略大小写
	if re, err := regexp2.Compile("(u(ltra).+hd|2160p|4k)", regexp2.IgnoreCase); err == nil {
		if m, _ := re.FindStringMatch(originName); m != nil {
			return true, strings.ReplaceAll(originName, m.String(), "")
		}
	}
	return false, originName
}

// 2. extractSuffixEpisode 从文件名中提取尾部集数信息
// 支持的格式包括:
// - 1-9, a-z (1-2位数字)
// - part1
// - ipz.A
// - CD1
// - NOP019B.HD.wmv
func extractSuffixEpisode(originName string) (episode string, name string) {
	// 初始化返回值
	name = originName

	// 尝试匹配尾部数字(1-2位)
	// (?<!\d) 是零宽负向后行断言，确保数字前面不是数字
	// \d{1,2}$ 匹配1-2位数字且在字符串末尾
	numberPattern := regexp2.MustCompile(`(?<!\d)\d{1,2}$`, regexp2.None)

	if matches, _ := numberPattern.FindStringMatch(originName); matches != nil {
		// 提取匹配到的数字
		episode = matches.String()
		// 从原始名称中移除匹配到的数字
		name, _ = numberPattern.Replace(originName, "", -1, -1)
		return
	}

	// 如果没有匹配到数字，尝试匹配尾部单个字母
	// (?<![a-zA-Z]) 是零宽负向后行断言，确保字母前面不是字母
	// [a-zA-Z]$ 匹配单个字母且在字符串末尾
	alphaPattern := regexp2.MustCompile(`(?<![a-zA-Z])[a-zA-Z]$`, regexp2.None)

	if matches, _ := alphaPattern.FindStringMatch(originName); matches != nil {
		// 提取匹配到的字母并转换为大写
		episode = strings.ToUpper(matches.String())
		// 从原始名称中移除匹配到的字母
		name, _ = alphaPattern.Replace(originName, "", -1, -1)
		return
	}

	return
}

// CodeInfo 存储提取的番号信息
type CodeInfo struct {
	Code         string // 番号
	Episode      string // 集数
	IsUncensored bool   // 是否无码
	IsCracked    bool   // 是否破解
	IsLeaked     bool   // 是否泄露
	IsCnSubs     bool   // 是否中文字幕
}

// ExtractCode 从文件名中提取番号信息
// 处理过程包括:
// 1. 识别特殊标记(中文字幕、无码、泄露、破解等)
// 2. 移除这些标记以减少干扰
// 3. 按照特定规则匹配番号
// 4. 处理番号格式(添加连字符等)
func extractCode(originName string) *CodeInfo {
	// 初始化结果
	result := &CodeInfo{}

	// 编译正则表达式
	chineseSubsPattern := regexp2.MustCompile(
		`(?<![a-zA-Z])(ch\b)|(中?文?字幕?)`,
		regexp2.IgnoreCase,
	)
	uncensoredPattern := regexp2.MustCompile(
		`(unc?e?n?s?o?r?e?d?)|(无码)`,
		regexp2.IgnoreCase,
	)
	leakPattern := regexp2.MustCompile(
		`(leak(ed)?)|(泄漏)|(流出)`,
		regexp2.IgnoreCase,
	)
	crackPattern := regexp2.MustCompile(
		`(crack(ed)?)|(破解)`,
		regexp2.IgnoreCase,
	)

	// 检查特殊标记
	result.IsCnSubs = utils.HasMatch(chineseSubsPattern, originName) //这里不提取C/c ,后面提取分集也会提取
	result.IsUncensored = utils.HasMatch(uncensoredPattern, originName)
	result.IsLeaked = utils.HasMatch(leakPattern, originName)
	result.IsCracked = utils.HasMatch(crackPattern, originName)

	// 移除匹配到的标记，减少干扰
	cleanName := utils.RemoveMatches(originName, []*regexp2.Regexp{
		chineseSubsPattern,
		uncensoredPattern,
		leakPattern,
		crackPattern,
	})

	// 尝试使用特殊规则匹配番号
	customizedCode, customizedEpisode, customizedIsCnSub := func(name string) (string, string, bool) {
		// 定义特殊规则映射
		specialRules := map[string]func(string) (string, error){
			`tokyo.*hot`: func(s string) (string, error) {
				pattern := regexp2.MustCompile(`(cz|gedo|k|n|red-|se)\d{2,4}`, regexp2.IgnoreCase)
				if m, _ := pattern.FindStringMatch(s); m != nil {
					return m.String(), nil
				}
				return "", fmt.Errorf("no match")
			},
			`carib|1pon|mura|paco`: func(s string) (string, error) {
				pattern := regexp2.MustCompile(`\d{6}(-|_)\d{3}`, regexp2.IgnoreCase)
				if m, _ := pattern.FindStringMatch(s); m != nil {
					return strings.ReplaceAll(m.String(), "-", "_"), nil
				}
				return "", fmt.Errorf("no match")
			},
			// ... 其他特殊规则实现
		}
		matchedEpisode := ""
		matchedIsCnSub := false
		matchedCode := ""
		// 尝试每个特殊规则
		for pattern, extractor := range specialRules {
			if matched, _ := regexp2.MustCompile(pattern, regexp2.IgnoreCase).MatchString(name); matched {
				if code, err := extractor(name); err == nil {
					// 提取集数
					matchedEpisode, matchedIsCnSub = extractEpisodeAndIsCnSubBehindCode(name, code)
					matchedCode = code
					break

				}
			}
		}

		return matchedCode, matchedEpisode, matchedIsCnSub

	}(cleanName)

	if customizedCode != "" {
		result.Code = customizedCode
		result.Episode = customizedEpisode
		result.IsCnSubs = result.IsCnSubs || customizedIsCnSub
		return result
	} else {
		//如果特殊规则不生效,则使用通用规则匹配番号
		generalCode, generalEpisode, generalIsCnSub := func(name string) (string, string, bool) {
			// 匹配通用番号格式
			pattern := regexp2.MustCompile(
				`(?:\d{2,}[-_]\d{2,})|(?:[A-Z]+[-_]?[A-Z]*\d{2,})+`,
				regexp2.IgnoreCase,
			)

			// match, _ := pattern.FindStringMatch(name)
			matches, _ := utils.FindAllMatches(pattern, name)
			if len(matches) == 0 {
				debugLogger.Shared().Debug("通用匹配 未匹配")
				return "", "", false
			}
			// TODO: 少数遇到匹配到多个番号的情况,比如 "Max-Girls-15-Haruka-Ito-Marimi-Natsusaki-Arisa-Kuroki-Aino-Kishi-Cecil-Fujisaki-Rio-[XV723]"
			// 暂时按照匹配长度排序,取最短长度的作为结果. 等待更好的算法,或者接入AI.
			sort.Slice(matches, func(i, j int) bool {
				return len(matches[i]) < len(matches[j])
			})
			// 暂时取最短的字数的
			code := matches[0]
			// 处理没有连字符的情况
			if !strings.Contains(code, "-") {
				// 提取字母+数字格式
				alphaNumPattern := regexp2.MustCompile(`[a-zA-Z]+\d{2,}`, regexp2.IgnoreCase)
				if m, _ := alphaNumPattern.FindStringMatch(code); m != nil {
					code = m.String()

					// HEYZO特殊处理
					if !strings.Contains(strings.ToLower(code), "heyzo") {
						// 添加连字符
						formatPattern := regexp2.MustCompile(
							`([a-zA-Z]{2,})(?:0*?)(\d{2,})`,
							regexp2.IgnoreCase,
						)
						code, _ = formatPattern.Replace(code, "$1-$2", -1, -1)
					}
				}
			}

			// 标准化处理
			standardPattern := regexp2.MustCompile(
				`([a-zA-Z]{2,})-(?:0*)(\d{3,})`,
				regexp2.IgnoreCase,
			)
			if match, _ := standardPattern.FindStringMatch(code); match != nil &&
				!strings.Contains(strings.ToLower(code), "heyzo") {
				code = match.Groups()[1].String() + "-" + match.Groups()[2].String()
			}

			// 提取集数
			episode, isCnSub := extractEpisodeAndIsCnSubBehindCode(name, code)

			return code, episode, isCnSub
		}(cleanName)
		if generalCode != "" {
			result.Code = generalCode
			result.Episode = generalEpisode

		} else {
		}
		result.IsCnSubs = result.IsCnSubs || generalIsCnSub
		return result
	}

}

// extractEpisodeBehindCode 从指定代码后面提取集数信息
// 支持的格式:
// - 字母(A-Z)
// - 数字(单个数字)
// 例如: "ABC-A" 或 "ABC-1" 或 "ABC1"
func extractEpisodeAndIsCnSubBehindCode(originName, code string) (string, bool) {
	// 构建正则表达式模式
	// (?i) 使整个模式不区分大小写
	// (?P<alpha>(\b[A-Z]\b)) 命名捕获组 "alpha"，匹配单个字母
	// \w*(?P<num>\d(?!\d)) 命名捕获组 "num"，匹配后面不跟数字的单个数字
	pattern := regexp2.MustCompile(
		fmt.Sprintf(`(?i)(?<=%s)(?:-(\b[A-Z]\b))?.*?(?:-\w*(\d)(?!\d))`, regexp2.Escape(code)),
		regexp2.IgnoreCase,
	)

	// m, e := utils.FindAllMatches(pattern, originName)
	// debugLogger.Shared().Debugf("find all string match: %v, %v", m, e)
	// 查找匹配
	match, err := pattern.FindStringMatch(originName)
	// debugLogger.Shared().Debugf("find string match: %v, %v", match, err)
	// matches, err := utils.FindAllStringSubmatch(pattern, originName)

	if err != nil || match == nil {
		return "", false
	}

	// 检查捕获组
	// Groups[0] 是整个匹配
	// Groups[1] 是字母组 (alpha)
	// Groups[2] 是数字组 (num)
	if match.GroupCount() < 3 {
		return "", false
	}
	// episodeDigital, isFindEpisodeDigital := lo.Find(matches, func(e utils.Match) bool {
	// 	return e.Groups[2] != ""
	// })
	// return episodeDigital.FullMatch, isFindEpisodeDigital
	// 优先使用数字组，如果没有则使用字母组
	var episodeDigital, episodeAlpha string
	if match.Groups()[2].String() != "" {
		episodeDigital = match.Groups()[2].String()
	}
	if match.Groups()[1].String() != "" {
		episodeAlpha = match.Groups()[1].String()
		episodeAlpha = strings.ToUpper(episodeAlpha)
	}

	if episodeDigital != "" && episodeAlpha == "C" { //特殊情况: 有数字集 又有字母集C，则说明其中一个C表示中文,另一个是 集
		return episodeDigital, true
	} else {
		return lo.CoalesceOrEmpty(episodeDigital, episodeAlpha), false
	}

}

// go:deprecate: cleanFileName函数已经做了清楚工作.
// GetCleanID 将番号中`-`, `_` 进行移除
func GetCleanID(str string) string {
	sb := strings.Builder{}
	for _, c := range str {
		if c == '-' || c == '_' {
			continue
		}
		sb.WriteRune(c)
	}
	return sb.String()
}
