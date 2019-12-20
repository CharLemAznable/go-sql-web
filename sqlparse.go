package main

import (
    "github.com/bingoohuang/gou/str"
    "net/http"
    "strings"

    "github.com/xwb1989/sqlparser"
)

func parseSql(querySql, dbDriverName, dbDataSource string) (string, []string) {
    var tableName string
    var primaryKeys []string
    firstWord := strings.ToUpper(str.FirstWord(querySql))
    switch firstWord {
    case "SELECT":
        if "goracle" == dbDriverName {
            querySql = strings.ReplaceAll(querySql, `"`, "`")
        }
        sqlParseResult, err := sqlparser.Parse(querySql)
        if err == nil {
            tableName = findSingleTableName(sqlParseResult)
            if tableName != "" {
                tableName, primaryKeys = findTablePrimaryKeys(tableName, dbDriverName, dbDataSource)
            }
        }
    default:
    }

    return tableName, primaryKeys
}

func writeAuthOk(r *http.Request) bool {
    return len(appConfig.WriteAuthUserNames) == 0 ||
        str.IndexOf(loginedUserName(r), appConfig.WriteAuthUserNames...) >= 0
}

func findPrimaryKeysIndex(tableName string, primaryKeys, headers []string) []int {
    primaryKeysIndex := make([]int, 0)
    if tableName == "" {
        return primaryKeysIndex
    }

    for _, primaryKey := range primaryKeys {
        for headerIndex, header := range headers {
            if primaryKey == header {
                primaryKeysIndex = append(primaryKeysIndex, headerIndex)
                break
            }
        }
    }

    return primaryKeysIndex
}

func findTablePrimaryKeys(tableName string, dbDriverName, dbDataSource string) (string, []string) {
    qualifiedName := tableName
    _, qual, _, _, err, _ := executeQuery(
        SqlOf(dbDriverName).QualifyTable(tableName),
        dbDriverName, dbDataSource, 0)
    if nil == err && len(qual) > 0 {
        qualifiedName = qual[0][1]
    }

    primaryKeys := make([]string, 0)
    _, data, _, _, err, _ := executeQuery(
        SqlOf(dbDriverName).DescribeTable(tableName),
        dbDriverName, dbDataSource, 0)
    if nil == err {
        for _, row := range data {
            if row[4] == "PRI" { // primary keys
                fieldName := row[1]
                primaryKeys = append(primaryKeys, fieldName)
            }
        }
    }

    return qualifiedName, primaryKeys
}

func findSingleTableName(sqlParseResult sqlparser.Statement) string {
    selectSql, _ := sqlParseResult.(*sqlparser.Select)
    if selectSql == nil || len(selectSql.From) != 1 {
        return ""
    }

    aliasTableExpr, ok := selectSql.From[0].(*sqlparser.AliasedTableExpr)
    if !ok {
        return ""
    }

    tableName, ok := aliasTableExpr.Expr.(sqlparser.TableName)
    if !ok {
        return ""
    }

    if tableName.Qualifier.IsEmpty() {
        return tableName.Name.String()
    }
    return tableName.Qualifier.String() + "." + tableName.Name.String()
}
