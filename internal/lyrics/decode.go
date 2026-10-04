package lyrics

import (
	"bytes"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/unicode"
)

// Decode 把歌词文件的字节转成字符串。
// 先看有没有 BOM（UTF-8 / UTF-16），再看是不是合法的 UTF-8，都不是就按 GBK 解码
// （很多中文 LRC 文件是用 Windows 记事本以 GBK 保存的）。
func Decode(data []byte) string {
	switch {
	case bytes.HasPrefix(data, []byte{0xEF, 0xBB, 0xBF}):
		return string(data[3:])
	case bytes.HasPrefix(data, []byte{0xFF, 0xFE}), bytes.HasPrefix(data, []byte{0xFE, 0xFF}):
		// UseBOM：根据 BOM 判断大小端
		decoded, err := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder().Bytes(data)
		if err == nil {
			return string(decoded)
		}
	}
	if utf8.Valid(data) {
		return string(data)
	}
	decoded, err := simplifiedchinese.GBK.NewDecoder().Bytes(data)
	if err != nil {
		return string(data) // 实在解不了就原样返回，至少能看到一部分
	}
	return string(decoded)
}
