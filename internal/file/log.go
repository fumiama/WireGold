package file

import (
	"encoding/hex"
	"runtime"
	"strings"
)

func Header() string {
	file, fn := fileFuncName(2)
	sb := strings.Builder{}
	sb.WriteString("[")
	sb.WriteString(file)
	sb.WriteString("] ")
	sb.WriteString(fn)
	return sb.String()
}

func fileFuncName(skip int) (string, string) {
	pc, file, _, ok := runtime.Caller(skip)
	if !ok {
		return "unknown", "unknown"
	}
	fn := runtime.FuncForPC(pc).Name()
	i := strings.LastIndex(fn, "/")
	fn = fn[i+1:]
	i = strings.LastIndex(file, "/")
	if i < 0 {
		i = strings.LastIndex(file, "\\")
		if i < 0 {
			return file, fn
		}
	}
	nm := file[i+1:]
	if len(nm) == 0 {
		return file, fn
	}
	i = strings.LastIndex(nm, ".")
	if i <= 0 {
		return nm, fn
	}
	return nm[:i], fn
}

func ToLimitHexString(data []byte, bound int) string {
	endl := "..."
	if len(data) < bound {
		bound = len(data)
		endl = "."
	}
	return hex.EncodeToString(data[:bound]) + endl
}
