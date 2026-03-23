package utils

import (
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// 格式化单个值
func formatValue(v interface{}) string {
	if v == nil {
		return "NULL"
	}

	switch val := v.(type) {
	case string:
		return "'" + escape(val) + "'"

	case []byte:
		return "'" + escape(string(val)) + "'"

	case time.Time:
		return "'" + val.Format("2006-01-02 15:04:05.000") + "'"

	case bool:
		if val {
			return "1"
		}
		return "0"

	// 👉 JSON / map / struct 自动转 JSON
	default:
		rv := reflect.ValueOf(v)
		// slice（但排除 []byte）
		if rv.Kind() == reflect.Slice {
			var arr []string
			for i := 0; i < rv.Len(); i++ {
				arr = append(arr, formatValue(rv.Index(i).Interface()))
			}
			return "(" + strings.Join(arr, ",") + ")"
		}
		// map / struct → JSON
		if rv.Kind() == reflect.Map || rv.Kind() == reflect.Struct {
			b, _ := json.Marshal(v)
			return "'" + escape(string(b)) + "'"
		}
		return fmt.Sprintf("%v", val)
	}
}

// 转义单引号
func escape(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// 核心函数
func BuildSQL(sql string, vars []interface{}) string {
	// ? 占位符（MySQL）
	if strings.Contains(sql, "?") {
		for _, v := range vars {
			sql = strings.Replace(sql, "?", formatValue(v), 1)
		}
		return sql
	}
	// $1 $2（Postgres）
	re := regexp.MustCompile(`\$\d+`)
	return re.ReplaceAllStringFunc(sql, func(m string) string {
		idx, _ := strconv.Atoi(m[1:])
		if idx-1 < len(vars) {
			return formatValue(vars[idx-1])
		}
		return m
	})
}
func DebugSQL(db *gorm.DB) string {
	stmt := db.Statement
	return BuildSQL(stmt.SQL.String(), stmt.Vars)
}

// ANSI 颜色
const (
	ColorReset = "\033[0m"
	ColorBlue  = "\033[34m" // 关键字
	ColorCyan  = "\033[36m" // 字段
	ColorGreen = "\033[32m" // 字符串
	ColorRed   = "\033[31m" // 慢SQL
)

// SQL关键字高亮
func highlightSQL(sql string) string {
	keywords := []string{
		"SELECT", "UPDATE", "INSERT", "DELETE",
		"FROM", "WHERE", "SET", "VALUES",
		"ORDER BY", "GROUP BY", "LIMIT",
	}

	for _, kw := range keywords {
		sql = regexp.MustCompile(`(?i)\b`+kw+`\b`).
			ReplaceAllString(sql, ColorBlue+kw+ColorReset)
	}

	// 字符串高亮
	sql = regexp.MustCompile(`'[^']*'`).
		ReplaceAllStringFunc(sql, func(s string) string {
			return ColorGreen + s + ColorReset
		})

	return sql
}

// 简单格式化（换行）
func formatSQL(sql string) string {
	replacer := strings.NewReplacer(
		" SET ", "\nSET ",
		" WHERE ", "\nWHERE ",
		" FROM ", "\nFROM ",
		" ORDER BY ", "\nORDER BY ",
	)
	return replacer.Replace(sql)
}

// Debug SQL（核心入口）高亮显示输出
/** 示例：
stmt := db.Session(&gorm.Session{DryRun: true}).
	Where("id IN ?", []int{1, 2, 3}).
	Find(&users).Statement
start := time.Now()
// 模拟执行耗时
time.Sleep(120 * time.Millisecond)
cost := time.Since(start)
fmt.Println(utils.DebugSQL(stmt, cost))
*/
func DebugSQLHighlight(stmt *gorm.Statement, cost time.Duration) string {
	raw := BuildSQL(stmt.SQL.String(), stmt.Vars)
	formatted := formatSQL(raw)
	colored := highlightSQL(formatted)
	// 慢SQL标红
	if cost > 200*time.Millisecond {
		return fmt.Sprintf("%s[SLOW SQL %v]\n%s%s", ColorRed, cost, colored, ColorReset)
	}
	return fmt.Sprintf("[SQL %v]\n%s", cost, colored)
}

// 高亮打印慢SQL
func LogSQL(db *gorm.DB, start time.Time) string {
	stmt := db.Statement
	cost := time.Since(start)
	return DebugSQLHighlight(stmt, cost)
}

/*
*

	WithSQLLog(db, func(tx *gorm.DB) *gorm.DB {
		return tx.Where("id = ?", 1).First(&user)
	})
*/
func WithSQLLog(db *gorm.DB, fn func(tx *gorm.DB) *gorm.DB) *gorm.DB {
	start := time.Now()
	tx := fn(db)
	LogSQL(tx, start)
	return tx
}

/*
注册回调
示例：
初始化时调用 RegisterSQLLogger(db)  之后你写任何sql：db.First(&user, 1)
一个关键坑（必须知道）:
tx := db.Where(...)
LogSQL(tx, start) // 没执行
❌ 是拿不到 SQL 的！
正确：
tx := db.Where(...).Find(&users)
LogSQL(tx, start)
*/
func RegisterSQLLogger(db *gorm.DB) {
	db.Callback().Query().After("gorm:query").Register("log_sql", func(tx *gorm.DB) {
		LogSQL(tx, time.Now()) // ⚠️ 这里没法精确拿 start，只能近似
	})

	db.Callback().Create().After("gorm:create").Register("log_sql", func(tx *gorm.DB) {
		LogSQL(tx, time.Now())
	})

	db.Callback().Update().After("gorm:update").Register("log_sql", func(tx *gorm.DB) {
		LogSQL(tx, time.Now())
	})

	db.Callback().Delete().After("gorm:delete").Register("log_sql", func(tx *gorm.DB) {
		LogSQL(tx, time.Now())
	})
}
