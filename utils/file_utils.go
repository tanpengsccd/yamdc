package utils

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"
	"yamdc/debugLogger"

	"github.com/spf13/afero"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// FileManager 提供文件操作的抽象层
type FileManager struct {
	fs afero.Fs
}

// NewFileManager 创建新的文件管理器
func NewFileManager() *FileManager {
	return &FileManager{
		fs: afero.NewOsFs(),
	}
}

// Move 提供可靠的文件移动功能
func (fm *FileManager) Move(srcFile, dstFile string) error {

	// 解码路径
	decodedPath, err := fm.decodePath(srcFile)
	if err != nil {
		return fmt.Errorf("error decoding path: %w", err)
	}
	// 首先规范化路径
	srcFile = fm.NormalizePathForPlatform(decodedPath)
	dstFile = fm.NormalizePathForPlatform(dstFile)

	// 确保源文件存在
	exists, err := afero.Exists(fm.fs, srcFile)
	if err != nil {
		return fmt.Errorf("error checking source file: %w", err)
	}
	if !exists {
		errinfo := fm.debugPathInfo(srcFile)
		debugLogger.Shared().Sugar().Errorf("source file does not exist: %s\n%s", srcFile, errinfo)
		return fmt.Errorf("source file does not exist: %s", srcFile)
	}

	// 确保目标目录存在
	dstDir := filepath.Dir(dstFile)
	err = fm.fs.MkdirAll(dstDir, 0755)
	if err != nil {
		return fmt.Errorf("error creating destination directory: %w", err)
	}

	// 尝试直接重命名
	err = fm.fs.Rename(srcFile, dstFile)
	if err == nil {
		return nil
	}

	// 如果重命名失败，尝试复制后删除的方式
	if err != nil && (strings.Contains(err.Error(), "invalid cross-device link") ||
		strings.Contains(err.Error(), "cross-device link") ||
		strings.Contains(err.Error(), "no such file")) {

		return fm.MoveByCopy(srcFile, dstFile)
	}

	return err
}

// moveByCopy 通过复制和删除来移动文件
func (fm *FileManager) MoveByCopy(srcFile, dstFile string) error {
	// 打开源文件
	srcHandle, err := fm.openSourceFile(srcFile)
	if err != nil {
		return fmt.Errorf("error opening source file: %w", err)
	}
	defer srcHandle.Close()

	// 创建目标文件
	dstHandle, err := fm.fs.Create(dstFile)
	if err != nil {
		return fmt.Errorf("error creating destination file: %w", err)
	}
	defer dstHandle.Close()

	// 复制文件内容
	_, err = io.Copy(dstHandle, srcHandle)
	if err != nil {
		// 如果复制失败，尝试清理目标文件
		dstHandle.Close()
		fm.fs.Remove(dstFile)
		return fmt.Errorf("error copying file: %w", err)
	}

	// 确保写入完成
	err = dstHandle.Close()
	if err != nil {
		return fmt.Errorf("error finalizing destination file: %w", err)
	}

	// 删除源文件
	err = fm.fs.Remove(srcFile)
	if err != nil {
		// 如果删除源文件失败，不要删除已复制的文件
		return fmt.Errorf("error removing source file: %w", err)
	}

	return nil
}

// 改进后的文件打开函数
func (fm *FileManager) openSourceFile(srcFile string) (afero.File, error) {
	// 新增原始路径检查
	if _, err := os.Stat(srcFile); err == nil {
		return fm.fs.Open(srcFile)
	}
	// 清理和规范化路径
	sanitizedPath := sanitizeFilePath(srcFile)
	normalizedPath := fm.NormalizePathForPlatform(sanitizedPath)

	// 首先检查文件是否存在
	exists, err := afero.Exists(fm.fs, normalizedPath)
	if err != nil {
		return nil, fmt.Errorf("error checking file existence: %w", err)
	}
	if !exists {
		// 如果文件不存在，尝试其他标准化形式
		alternatePath := fm.tryAlternateNormalization(normalizedPath)
		if alternatePath != "" {
			normalizedPath = alternatePath
		} else {
			return nil, fmt.Errorf("file not found: %s", srcFile)
		}
	}

	// 尝试打开文件
	srcHandle, err := fm.fs.Open(normalizedPath)
	if err != nil {
		// 提供更详细的错误信息
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found after existence check (race condition?): %s", normalizedPath)
		}
		if os.IsPermission(err) {
			return nil, fmt.Errorf("permission denied accessing file: %s", normalizedPath)
		}
		return nil, fmt.Errorf("error opening source file: %w", err)
	}

	return srcHandle, nil
}

// tryAlternateNormalization 尝试其他 Unicode 标准化形式
func (fm *FileManager) tryAlternateNormalization(path string) string {
	// 尝试另一种标准化形式
	var alternatePath string
	if runtime.GOOS == "darwin" { // macOS 使用 NFC 标准
		alternatePath = norm.NFC.String(path)
	} else { // 其他系统Linux Windows 使用 NFD 标准
		alternatePath = norm.NFD.String(path)
	}

	exists, _ := afero.Exists(fm.fs, alternatePath)
	if exists {
		return alternatePath
	}
	return ""
}

// readDirSafely 安全地读取目录内容
func (fm *FileManager) ReadDirSafely(dirPath string) ([]fs.DirEntry, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	return entries, nil
}

// normalizePathForPlatform 根据不同平台规范化文件路径
func (fm *FileManager) NormalizePathForPlatform(path string) string {
	path = filepath.Clean(path)

	// 优先尝试NFC标准化
	path = norm.NFC.String(path)

	// macOS额外尝试NFD形式
	if runtime.GOOS == "darwin" {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			nfdPath := norm.NFD.String(path)
			if _, err := os.Stat(nfdPath); err == nil {
				return nfdPath
			}
		}
	}
	// 处理macOS长路径
	if runtime.GOOS == "darwin" && len(path) > 1024 {
		path = "/private" + path
	}
	return path
}

func (fm *FileManager) decodePath(path string) (string, error) {
	// 如果路径已经是有效的 UTF-8，直接返回
	if utf8.ValidString(path) {
		return path, nil
	}

	// 尝试不同的编码转换
	decoders := []transform.Transformer{
		japanese.ShiftJIS.NewDecoder(),
		japanese.EUCJP.NewDecoder(),
		japanese.ISO2022JP.NewDecoder(),
	}

	for _, decoder := range decoders {
		decoded, err := fm.decodeString(path, decoder)
		if err == nil && utf8.ValidString(decoded) {
			return decoded, nil
		}
	}

	return "", fmt.Errorf("failed to decode path: %s", path)
}

// decodeString 解码字符串
func (fm *FileManager) decodeString(s string, decoder transform.Transformer) (string, error) {
	decoded, _, err := transform.String(decoder, s)
	return decoded, err
}

// 首先定义一个helper函数来清理和验证文件路径
func sanitizeFilePath(path string) string {
	// 使用filepath.Clean来规范化路径
	cleanPath := filepath.Clean(path)
	return cleanPath
	// // 将路径分隔符统一为系统分隔符
	// // return filepath.FromSlash(cleanPath)
	// replacer := strings.NewReplacer(
	// 	"(", "\\(",
	// 	")", "\\)",
	// 	" ", "\\ ",
	// )
	// return replacer.Replace(cleanPath)

}

// isDir 检查指定路径是否为目录
func (fm *FileManager) IsDir(path string) (bool, error) {
	fi, err := fm.fs.Stat(path)
	if err != nil {
		return false, err

	} else {
		return fi.IsDir(), nil
	}
}
func (fm *FileManager) IsExist(path string) (bool, error) {
	return afero.Exists(fm.fs, path)

}
func (fm *FileManager) debugPathInfo(path string) string {
	// 新增系统级访问测试
	sysInfo := "\nSystem access test:\n"
	if _, err := os.Open(path); err != nil {
		sysInfo += fmt.Sprintf("os.Open error: %v\n", err)
	} else {
		sysInfo += "os.Open success\n"
	}

	var info strings.Builder

	info.WriteString(fmt.Sprintf("Original path: %s\n", path))
	info.WriteString(fmt.Sprintf("Clean path: %s\n", filepath.Clean(path)))
	info.WriteString(fmt.Sprintf("NFC form: %s\n", norm.NFC.String(path)))
	info.WriteString(fmt.Sprintf("NFD form: %s\n", norm.NFD.String(path)))

	// 检查文件是否存在（使用不同的标准化形式）
	forms := map[string]string{
		"Original": path,
		"Clean":    filepath.Clean(path),
		"NFC":      norm.NFC.String(path),
		"NFD":      norm.NFD.String(path),
	}

	for form, p := range forms {
		exists, _ := afero.Exists(fm.fs, p)
		info.WriteString(fmt.Sprintf("%s form exists: %v\n", form, exists))
	}

	return info.String()
}

// -----------------
func GetExtName(f string, def string) string {
	if v := filepath.Ext(f); v != "" {
		return v
	}
	return def
}
