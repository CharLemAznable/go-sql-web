package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/bingoohuang/gonet"
	"github.com/bingoohuang/gou/str"
	"github.com/bingoohuang/sqlmore"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type QueryResult struct {
	Headers             []string
	Rows                [][]string
	TableColumns        map[string][]string
	Error               string
	ExecutionTime       string
	CostTime            string
	DriverName          string
	DatabaseName        string
	TableName           string
	PrimaryKeysIndex    []int
	Msg                 string
	Tid                 string
	MultipleTenantsExec []string
}

func serveTablesByColumn(w http.ResponseWriter, req *http.Request) {
	gonet.ContentTypeJSON(w)
	tid := strings.TrimSpace(req.FormValue("tid"))
	columnName := strings.TrimSpace(req.FormValue("columnName"))

	dbDriverName, dbDataSource, databaseName, err := selectDb(tid)
	if err != nil {
		http.Error(w, err.Error(), 405)
		return
	}

	querySql := "SELECT TABLE_NAME FROM INFORMATION_SCHEMA.COLUMNS " +
		"WHERE TABLE_SCHEMA NOT IN('information_schema','mysql','performance_schema') " +
		"AND COLUMN_NAME = '" + columnName + "'"

	_, rows, executionTime, costTime, err, msg := processSql(tid, querySql, dbDriverName, dbDataSource, 0)

	queryResult := struct {
		Rows          [][]string
		Error         string
		ExecutionTime string
		CostTime      string
		DatabaseName  string
		Msg           string
	}{
		Rows:          rows,
		Error:         str.Error(err),
		ExecutionTime: executionTime,
		CostTime:      costTime,
		DatabaseName:  databaseName,
		Msg:           msg,
	}

	_ = json.NewEncoder(w).Encode(queryResult)
}

func multipleTenantsQuery(w http.ResponseWriter, req *http.Request) {
	gonet.ContentTypeJSON(w)
	sqlString := strings.TrimFunc(req.FormValue("sql"), func(r rune) bool {
		return unicode.IsSpace(r) || r == ';'
	})

	sqls := sqlmore.SplitSqls(sqlString, ';')
	for _, subSql := range sqls {
		_, isQuery := sqlmore.IsQuerySQL(subSql)
		if isQuery {
			continue
		}
		if !writeAuthOk(req) {
			http.Error(w, "write auth required", 405)
			return
		}
	}

	tids := req.FormValue("multipleTenantIds")
	multipleTenantIds := strings.FieldsFunc(tids, func(c rune) bool { return c == ',' })

	tenantsSize := len(multipleTenantIds)
	resultChan := make(chan *QueryResult, tenantsSize)
	saveHistory(tids, sqlString)

	for _, tid := range multipleTenantIds {
		go executeSqlInTid(tid, resultChan, sqlString)
	}

	results := make([]*QueryResult, tenantsSize)
	for i := 0; i < tenantsSize; i++ {
		results[i] = <-resultChan
	}

	_ = json.NewEncoder(w).Encode(results)
}

func executeSqlInTid(tid string, resultChan chan *QueryResult, sqlString string) {
	dbDriverName, dbDataSource, databaseName, err := selectDbByTid(tid, appConfig.DriverName, appConfig.DataSource)
	if err != nil {
		resultChan <- &QueryResult{
			Error: str.Error(err),
			Tid:   tid,
		}
		return
	}

	db, err := sql.Open(dbDriverName, dbDataSource)
	if err != nil {
		resultChan <- &QueryResult{
			Error:        str.Error(err),
			DriverName:   dbDriverName,
			DatabaseName: databaseName,
			Tid:          tid,
		}

		return
	}
	defer func() { _ = db.Close() }()

	executionTime := time.Now().Format("2006-01-02 15:04:05.000")

	sqls := sqlmore.SplitSqls(sqlString, ';')
	sqlsLen := len(sqls)

	if sqlsLen == 1 {
		sqlResult := sqlmore.ExecSQL(db, sqls[0], 0, "(null)")
		msg := ""
		if !sqlResult.IsQuerySQL {
			msg = strconv.FormatInt(sqlResult.RowsAffected, 10) + " rows were affected"
		}
		result := QueryResult{
			Headers:       sqlResult.Headers,
			Rows:          sqlResult.Rows,
			Error:         str.Error(sqlResult.Error),
			ExecutionTime: executionTime,
			CostTime:      sqlResult.CostTime.String(),
			DriverName:    dbDriverName,
			DatabaseName:  databaseName,
			Tid:           tid,
			Msg:           msg,
		}
		resultChan <- &result

		return
	}

	querySqlMixed := false
	if sqlsLen > 1 {
		for _, oneSql := range sqls {
			_, isQuery := sqlmore.IsQuerySQL(oneSql)
			if isQuery {
				querySqlMixed = true
				break
			}
		}
	}

	if querySqlMixed {
		resultChan <- &QueryResult{
			Error:        "select sql should be executed one by one in single time",
			DriverName:   dbDriverName,
			DatabaseName: databaseName,
			Tid:          tid,
		}

		return
	}

	start := time.Now()
	msg := ""
	for _, oneSql := range sqls {
		sqlResult := sqlmore.ExecSQL(db, oneSql, 0, "(null)")
		if msg != "" {
			msg += "\n"
		}
		if sqlResult.Error != nil {
			msg += sqlResult.Error.Error()
		} else {
			msg += strconv.FormatInt(sqlResult.RowsAffected, 10) + " rows affected"
		}
	}

	resultChan <- &QueryResult{
		ExecutionTime: executionTime,
		CostTime:      time.Since(start).String(),
		DriverName:    dbDriverName,
		DatabaseName:  databaseName,
		Msg:           msg,
		Tid:           tid,
	}
}

func downloadColumn(w http.ResponseWriter, req *http.Request) {
	querySql := strings.TrimFunc(req.FormValue("sql"), func(r rune) bool {
		return unicode.IsSpace(r) || r == ';'
	})
	fileName := strings.TrimSpace(req.FormValue("fileName"))
	tid := strings.TrimSpace(req.FormValue("tid"))

	dn, ds, _, err := selectDb(tid)
	if err != nil {
		http.Error(w, err.Error(), 405)
		return
	}

	db, err := sql.Open(dn, ds)
	if err != nil {
		http.Error(w, err.Error(), 405)
		return
	}
	defer func() { _ = db.Close() }()

	rows, err := db.Query(querySql)
	if err != nil {
		http.Error(w, err.Error(), 405)
		return
	}
	columns, err := rows.Columns()
	if err != nil {
		http.Error(w, err.Error(), 405)
		return
	}

	columnCount := len(columns)
	if columnCount != 1 {
		http.Error(w, "only one column supported to download", 500)
		return
	}

	if !rows.Next() {
		http.Error(w, "Nothing to download", 500)
		return
	}

	// Make a slice for the values
	values := make([]sql.RawBytes, columnCount)

	// rows.Scan wants '[]interface{}' as an argument, so we must copy the
	// references into such a slice
	// See http://code.google.com/p/go-wiki/wiki/InterfaceSlice for details
	scanArgs := make([]interface{}, columnCount)
	for i := range values {
		scanArgs[i] = &values[i]
	}

	err = rows.Scan(scanArgs...)
	if err != nil {
		http.Error(w, err.Error(), 405)
		return
	}

	fmt.Println(reflect.TypeOf(values[0]))

	// tell the browser the returned content should be downloaded
	w.Header().Add("Content-Disposition", "Attachment; filename="+fileName)
	http.ServeContent(w, req, fileName, time.Now(), bytes.NewReader([]byte(values[0])))
}

func serveQuery(w http.ResponseWriter, req *http.Request) {
	gonet.ContentTypeJSON(w)
	querySql := strings.TrimFunc(req.FormValue("sql"), func(r rune) bool {
		return unicode.IsSpace(r) || r == ';'
	})

	_, isQuery := sqlmore.IsQuerySQL(querySql)
	if !IsCodedSql(querySql) && !isQuery && !writeAuthOk(req) {
		http.Error(w, "write auth required", 405)
		return
	}

	tid := strings.TrimSpace(req.FormValue("tid"))
	withColumns := strings.TrimSpace(req.FormValue("withColumns"))
	maxRowsStr := strings.TrimSpace(req.FormValue("maxRows"))
	maxRows := 0
	if maxRowsStr != "" {
		maxRows, _ = strconv.Atoi(maxRowsStr)
	}

	if maxRows < appConfig.MaxQueryRows {
		maxRows = appConfig.MaxQueryRows
	}

	dn, ds, dbName, err := selectDb(tid)
	if err != nil {
		http.Error(w, err.Error(), 405)
		return
	}

	actualSql := querySql
	if IsCodedSql(querySql) {
		actualSql = SqlOf(dn).DecodeQuerySql(querySql)
	}
	if strings.HasPrefix(strings.ToUpper(actualSql), "DECLARE") {
		actualSql = actualSql + ";"
	}

	tableName, primaryKeys := parseSql(actualSql, dn, ds)
	headers, rows, execTime, costTime, err, msg := processActualSql(tid, querySql, actualSql, dn, ds, maxRows)
	primaryKeysIndex := findPrimaryKeysIndex(tableName, primaryKeys, headers)

	queryResult := QueryResult{
		Headers:             headers,
		Rows:                rows,
		Error:               str.Error(err),
		ExecutionTime:       execTime,
		CostTime:            costTime,
		DriverName:          dn,
		DatabaseName:        dbName,
		TableName:           tableName,
		PrimaryKeysIndex:    primaryKeysIndex,
		Msg:                 msg,
		MultipleTenantsExec: appConfig.MultipleTenantsExecConfig[strings.ToUpper(tableName)],
	}

	if "true" == withColumns {
		tableColumns := make(map[string][]string)
		_, colRows, _, _, _, _ := executeQuery(SqlOf(dn).TableColumnsSql(dbName), dn, ds, 0)

		tableName := ""
		var columns []string

		for _, row := range colRows {
			if tableName != row[1] {
				if tableName != "" {
					tableColumns[tableName] = columns
					columns = make([]string, 0)
				}
				tableName = row[1]
			}

			columns = append(columns, row[2], row[3], row[4], row[5], row[6], row[7])
		}

		if tableName != "" {
			tableColumns[tableName] = columns
		}

		_, tableRows, _, _, _, _ := executeQuery(SqlOf(dn).TableCommentSql(dbName), dn, ds, 0)
		for _, row := range tableRows {
			tblName := row[1]
			_, ok := tableColumns[tblName]
			if ok {
				tableCommentCols := make([]string, 0)
				tableCommentCols = append(tableCommentCols, row[2])
				tableColumns[tblName+`_TABLE_COMMENT`] = tableCommentCols
			}
		}

		queryResult.TableColumns = tableColumns
	}

	_ = json.NewEncoder(w).Encode(queryResult)
}

func processSql(tid, querySql, dbDriverName, dbDataSource string, max int) ([]string, [][]string, string, string, error, string) {
	isShowHistory := strings.EqualFold("show history", querySql)
	if isShowHistory {
		return showHistory()
	}

	saveHistory(tid, querySql)
	return executeQuery(querySql, dbDriverName, dbDataSource, max)
}

func processActualSql(tid, querySql, actualSql, dbDriverName, dbDataSource string, max int) ([]string, [][]string, string, string, error, string) {
	isShowHistory := strings.EqualFold("show history", actualSql)
	if isShowHistory {
		return showHistory()
	}

	saveHistory(tid, querySql)
	return executeQuery(actualSql, dbDriverName, dbDataSource, max)
}
