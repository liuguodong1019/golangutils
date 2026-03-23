package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const batchSize = 2000

func main() {

	host := flag.String("host", "127.0.0.1", "mysql host")
	port := flag.Int("port", 3306, "mysql port")
	user := flag.String("user", "root", "mysql user")
	pass := flag.String("pass", "", "mysql password")
	dbName := flag.String("db", "", "database name")
	tables := flag.String("tables", "", "table list split by comma")
	output := flag.String("out", "dump.sql", "output file")

	flag.Parse()

	if *dbName == "" {
		panic("db name required")
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4",
		*user, *pass, *host, *port, *dbName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}

	var tableList []string

	if *tables != "" {
		tableList = strings.Split(*tables, ",")
	} else {
		tableList = getAllTables(db)
	}

	file, err := os.Create(*output)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	file.WriteString("SET FOREIGN_KEY_CHECKS=0;\n\n")

	for _, table := range tableList {

		fmt.Println("export:", table)

		exportTableStructure(db, file, table)
		exportTableData(db, file, table)
	}

	file.WriteString("\nSET FOREIGN_KEY_CHECKS=1;\n")

	fmt.Println("done")
}

func getAllTables(db *sql.DB) []string {

	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var tables []string
	var table string

	for rows.Next() {
		rows.Scan(&table)
		tables = append(tables, table)
	}

	return tables
}

func exportTableStructure(db *sql.DB, file *os.File, table string) {

	var name, createSQL string

	err := db.QueryRow("SHOW CREATE TABLE `"+table+"`").Scan(&name, &createSQL)
	if err != nil {
		panic(err)
	}

	file.WriteString(createSQL + ";\n\n")
}

func exportTableData(db *sql.DB, file *os.File, table string) {

	rows, err := db.Query("SELECT * FROM `" + table + "`")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	cols, _ := rows.Columns()

	values := make([]interface{}, len(cols))
	valuePtrs := make([]interface{}, len(cols))

	for i := range values {
		valuePtrs[i] = &values[i]
	}

	insertPrefix := fmt.Sprintf("INSERT INTO `%s` VALUES ", table)

	count := 0
	var batch []string

	for rows.Next() {

		rows.Scan(valuePtrs...)

		var rowValues []string

		for _, v := range values {

			if v == nil {
				rowValues = append(rowValues, "NULL")
				continue
			}

			var val string

			switch vv := v.(type) {

			case []byte:
				val = string(vv)

			case string:
				val = vv

			case time.Time:
				val = vv.Format("2006-01-02 15:04:05")

			case bool:
				if vv {
					val = "1"
				} else {
					val = "0"
				}

			default:
				val = fmt.Sprintf("%v", vv)
			}

			val = strings.ReplaceAll(val, "\\", "\\\\")
			val = strings.ReplaceAll(val, "'", "\\'")

			rowValues = append(rowValues, "'"+val+"'")
		}

		batch = append(batch, "("+strings.Join(rowValues, ",")+")")

		count++

		if count%batchSize == 0 {

			sql := insertPrefix + strings.Join(batch, ",") + ";\n"

			file.WriteString(sql)

			batch = batch[:0]
		}
	}

	if len(batch) > 0 {

		sql := insertPrefix + strings.Join(batch, ",") + ";\n"

		file.WriteString(sql)
	}

	file.WriteString("\n\n")
}
