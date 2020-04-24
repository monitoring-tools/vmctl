package vm

import (
	"reflect"
	"unicode/utf8"
	"unsafe"
)

const hex = "0123456789abcdef"

func toByte(s string) []byte {
	header := (*reflect.StringHeader)(unsafe.Pointer(&s))
	slice := reflect.SliceHeader{
		Data: header.Data,
		Len:  header.Len,
		Cap:  header.Len,
	}

	return *(*[]byte)(unsafe.Pointer(&slice))
}

func FastEscape(out []byte, st string) []byte {
	out = append(out, '"')
	s := toByte(st)
	start := 0
	for i := 0; i < len(s); {
		if b := s[i]; b < utf8.RuneSelf {
			if 0x20 <= b && b != '\\' && b != '"' && b != '<' && b != '>' && b != '&' {
				i++
				continue
			}
			if start < i {
				out = append(out, s[start:i]...)
			}
			switch b {
			case '\\', '"':
				out = append(out, '\\')
				out = append(out, b)
			case '\n':
				out = append(out, "\\n"...)
			case '\r':
				out = append(out, "\\r"...)
			case '\t':
				out = append(out, "\\t"...)
			default:
				out = append(out, "\\u00"...)
				out = append(out, hex[b>>4])
				out = append(out, hex[b&0xf])
			}
			i++
			start = i
			continue
		}

		c, size := utf8.DecodeRune(s[i:])
		if c == utf8.RuneError && size == 1 {
			if start < i {
				out = append(out, s[start:i]...)
			}
			out = append(out, "\\ufffd"...)
			i += size
			start = i
			continue
		}

		if c == '\u2028' || c == '\u2029' {
			if start < i {
				out = append(out, s[start:i]...)
			}
			out = append(out, "\\u202"...)
			out = append(out, hex[c&0xF])
			i += size
			start = i
			continue
		}
		i += size
	}
	if start < len(s) {
		out = append(out, s[start:]...)
	}
	out = append(out, '"')

	return out
}
