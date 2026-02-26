package utils

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"net/http"
	"reflect"
)

// 判断结构体是否有值
func IsZeroStruct(v interface{}) bool {
	return reflect.ValueOf(v).IsZero()
}

func ToJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

// 判断是否有值
func HasValue[T any](v T) bool {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return false
	}

	switch rv.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice,
		reflect.Func, reflect.Interface, reflect.Chan:
		return !rv.IsNil()
	default:
		return true
	}
}

// 按值删除
func RemoveByValue[T comparable](s []T, v T) []T {
	for i, x := range s {
		if x == v {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}

// 按条件删除
/**
示例：
items = Filter(items, func(i Item) bool {
	return i.Value != ""
})
*/
func Filter[T any](s []T, keep func(T) bool) []T {
	n := 0
	for _, v := range s {
		if keep(v) {
			s[n] = v
			n++
		}
	}
	return s[:n]
}

func ParamsBind(req *http.Request, v any) error {
	ginContext := gin.Context{Request: req}
	b := binding.Default(ginContext.Request.Method, ginContext.ContentType())
	err := ginContext.ShouldBindWith(v, b)
	return err
}

// a 数据库里已有的 ID 集合
func DiffSliceInt[T comparable](a, b []T) (toInsert []T, toDelete []T) {
	setA := make(map[T]struct{}, len(a))
	setB := make(map[T]struct{}, len(b))

	for _, v := range a {
		setA[v] = struct{}{}
	}
	for _, v := range b {
		setB[v] = struct{}{}
	}

	// b 中有，但 a 中没有 → 插入
	for v := range setB {
		if _, ok := setA[v]; !ok {
			toInsert = append(toInsert, v)
		}
	}

	// a 中有，但 b 中没有 → 删除
	for v := range setA {
		if _, ok := setB[v]; !ok {
			toDelete = append(toDelete, v)
		}
	}

	return
}

// 交集
func Intersection[T comparable](a, b []T) []T {
	if len(a) == 0 || len(b) == 0 {
		return nil
	}
	m := make(map[T]struct{}, len(a))
	res := make([]T, 0)
	for _, v := range a {
		m[v] = struct{}{}
	}
	for _, v := range b {
		if _, ok := m[v]; ok {
			res = append(res, v)
			delete(m, v) // 防止重复加入
		}
	}
	return res
}

// 差集 在 A 里但不在 B 里,a = {1,2,3} b={2,4} 结果：{1,3}，将b中没有的元素返回
func Difference[T comparable](a, b []T) []T {
	m := make(map[T]struct{}, len(b))
	res := make([]T, 0)
	for _, v := range b {
		m[v] = struct{}{}
	}
	for _, v := range a {
		if _, ok := m[v]; !ok {
			res = append(res, v)
		}
	}
	return res
}
