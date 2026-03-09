package template

import (
	"context"
	"reflect"
	"text/template"

	"github.com/youcd/toolkit/log"
)

//nolint:modernize
var (
	funcMap = template.FuncMap{
		"typeOf": func(v interface{}) string {
			if v == nil {
				return "nil"
			}
			return reflect.TypeOf(v).Kind().String()
		},

		"isType": func(v interface{}, typeName string) bool {
			if v == nil {
				return false
			}
			return reflect.TypeOf(v).String() == typeName
		},

		"isKind": func(v interface{}, kind string) bool {
			if v == nil {
				return false
			}
			return reflect.TypeOf(v).Kind().String() == kind
		},

		// 具体类型判断函数
		"isString": func(v interface{}) bool { return getKind(v) == "string" },
		"isSlice":  func(v interface{}) bool { return getKind(v) == "slice" },
		"isArray":  func(v interface{}) bool { return getKind(v) == "array" },
		"isMap":    func(v interface{}) bool { return getKind(v) == "map" },
		"isInt": func(v interface{}) bool {
			k := getKind(v)
			return k == "int" || k == "int8" || k == "int16" || k == "int32" || k == "int64"
		},
		"isUint": func(v interface{}) bool {
			k := getKind(v)
			return k == "uint" || k == "uint8" || k == "uint16" || k == "uint32" || k == "uint64"
		},
		"isFloat": func(v interface{}) bool {
			k := getKind(v)
			return k == "float32" || k == "float64"
		},
		"isBool":   func(v interface{}) bool { return getKind(v) == "bool" },
		"toLetter": toLetter,
	}
)

// getKind 辅助函数
//
//nolint:all
func getKind(v interface{}) string {
	if v == nil {
		return "invalid"
	}
	k := reflect.TypeOf(v).Kind().String()
	log.WithCtx(context.Background()).Debug("getKind:", k)
	return k
}

// 自定义函数：数字转字母 (0->A, 1->B, ...)
func toLetter(i int) string {
	return string(rune('A' + i))
}
