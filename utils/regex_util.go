package utils

import "github.com/dlclark/regexp2"

// 找到所有匹配
func FindAllMatches(re *regexp2.Regexp, text string) ([]string, error) {
	var matches []string
	match, err := re.FindStringMatch(text)

	for match != nil && err == nil {
		matches = append(matches, match.String())
		match, err = re.FindNextMatch(match)
	}

	return matches, err
}

// 辅助函数: 检查是否有匹配
func HasMatch(pattern *regexp2.Regexp, text string) bool {
	match, _ := pattern.FindStringMatch(text)
	return match != nil
}

// 辅助函数: 移除所有匹配项
func RemoveMatches(text string, patterns []*regexp2.Regexp) string {
	result := text
	for _, pattern := range patterns {
		for {
			match, _ := pattern.FindStringMatch(result)
			if match == nil {
				break
			}
			result = result[:match.Group.Index] + result[match.Group.Index+match.Group.Length:]
		}
	}
	return result
}

// Match 结构体用于存储匹配信息
type Match struct {
	FullMatch string   // 完整匹配
	Groups    []string // 捕获组
}

// FindAllStringSubmatch 返回所有匹配及其捕获组信息
func FindAllStringSubmatch(re *regexp2.Regexp, s string) ([]Match, error) {
	var matches []Match
	m, err := re.FindStringMatch(s)

	for m != nil && err == nil {
		match := Match{
			FullMatch: m.String(),
			Groups:    make([]string, len(m.Groups())),
		}

		// 保存所有捕获组的信息
		for i, group := range m.Groups() {
			match.Groups[i] = group.String()
		}

		matches = append(matches, match)
		m, err = re.FindNextMatch(m)
	}

	return matches, err
}
