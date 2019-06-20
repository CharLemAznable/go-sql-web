package main

import (
	"database/sql"
	"errors"
	"log"
	"strconv"
	"time"

	"fmt"
	"github.com/bingoohuang/gou"

	_ "github.com/go-goracle/goracle"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func selectDb(tid string) (string, string, string, error) {
	if tid == "" || tid == "trr" {
		_, rows, _, _, err, _ := executeQuery(
			SqlOf(appConfig.DriverName).SelectDb(),
			appConfig.DriverName, appConfig.DataSource, 0)
		if err != nil {
			return "", "", "", err
		}

		return appConfig.DriverName, appConfig.DataSource, rows[0][1], nil
	}

	return selectDbByTid(tid, appConfig.DriverName, appConfig.DataSource)
}

func selectDbByTid(tid string, dn, ds string) (string, string, string, error) {
	_, data, _, _, err, _ := executeQuery(SqlOf(dn).SelectDbByTid(tid), dn, ds, 1)
	if err != nil {
		return "", "", "", err
	}

	if len(data) == 0 {
		return "", "", "", errors.New("no db found for tid:" + tid)
	} else if len(data) > 1 {
		log.Println("data", data)
		return "", "", "", errors.New("more than one db found")
	}

	row := data[0]
	return SqlOf(dn).SelectDbByTidResult(row)
}

func executeQuery(querySql, driverName, dataSource string, max int) (
	[]string /*header*/, [][]string, /*data*/
	string /*executionTime*/, string /*costTime*/, error, string /* msg */) {
	db, err := sql.Open(driverName, dataSource)
	if err != nil {
		return nil, nil, "", "", err, ""
	}
	defer func() { _ = db.Close() }()

	return query(db, querySql, max)
}

func query(db *sql.DB, query string, maxRows int) ([]string, [][]string, string, string, error, string) {
	executionTime := time.Now().Format("2006-01-02 15:04:05.000")

	sqlResult := gou.ExecuteSql(db, query, maxRows)
	data := addRowsSeq(&sqlResult)
	fmt.Println("IsQuerySql:", sqlResult.IsQuerySql)

	msg := ""
	if !sqlResult.IsQuerySql {
		msg = strconv.FormatInt(sqlResult.RowsAffected, 10) + " rows were affected"
	}

	return sqlResult.Headers, data, executionTime, sqlResult.CostTime.String(), sqlResult.Error, msg
}

func addRowsSeq(sqlResult *gou.ExecuteSqlResult) [][]string {
	data := make([][]string, 0)
	if sqlResult.Rows != nil {
		for index, row := range sqlResult.Rows {
			r := make([]string, len(row)+1)
			r[0] = strconv.Itoa(index + 1)
			for j, cell := range row {
				r[j+1] = cell
			}
			data = append(data, r)
		}
	}
	return data
}
