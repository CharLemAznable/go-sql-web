package main

import "C"
import (
    "strings"
)

type SqlTemp interface {
    SelectDb() string
    SelectDbByTid(tid string) string
    SelectDbByTidResult(row []string) (string, string, string, error)
    DescribeTable(tableName string) string
    DecodeQuerySql(querySql string) string
    TableColumnsSql(dbName string) string
    TableCommentSql(dbName string) string
}

// abstractSqlTemp

type abstractSqlTemp struct{}

func (t *abstractSqlTemp) SelectDbByTid(tid string) string {
    return `SELECT DB_USERNAME, DB_PASSWORD, PROXY_IP, PROXY_PORT, DB_NAME
        FROM TR_F_DB WHERE MERCHANT_ID = '` + tid + `'`
}

// goracle

type goracleTemp struct {
    abstractSqlTemp
}

func (t *goracleTemp) SelectDb() string {
    return "SELECT NAME FROM V$DATABASE"
}

func (t *goracleTemp) SelectDbByTidResult(row []string) (string, string, string, error) {
    // oracle://user:pass@//127.0.0.1:1521/db
    return "goracle", "oracle://" + row[1] + ":" + row[2] +
        "@//" + row[3] + ":" + row[4] + "/" +
        row[5], row[5], nil
}

func (t *goracleTemp) DescribeTable(tableName string) string {
    return `SELECT T.COLUMN_NAME AS "FIELD"
                  ,T.DATA_TYPE||DECODE (T.DATA_TYPE,
                                        'NUMBER', DECODE ('('
                                                          || NVL (TO_CHAR (DATA_PRECISION), '*')
                                                          || ','
                                                          || NVL (TO_CHAR (DATA_SCALE), '*')
                                                          || ')',
                                                          '(*,*)', NULL,
                                                          '(*,0)', '(38)',
                                                          '('
                                                          || NVL (TO_CHAR (DATA_PRECISION), '*')
                                                          || ','
                                                          || NVL (TO_CHAR (DATA_SCALE), '*')
                                                          || ')'),
                                        'FLOAT', '(' || DATA_PRECISION || ')',
                                        'DATE', NULL,
                                        'TIMESTAMP(6)', NULL,
                                        '(' || DATA_LENGTH || ')') AS "TYPE"
                  ,(SELECT CASE WHEN T.NULLABLE = 'N' THEN 'NO' ELSE 'YES' END FROM DUAL) AS "NULL"
                  ,(SELECT CASE WHEN T.COLUMN_NAME = M.COLUMN_NAME THEN 'PRI' ELSE '' END FROM DUAL) AS "KEY"
                  ,T.DATA_DEFAULT AS "DEFAULT"
                  ,C.COMMENTS AS "Comment"
              FROM USER_TAB_COLS T
              LEFT JOIN (
                   SELECT L.TABLE_NAME
                         ,L.COLUMN_NAME
                     FROM USER_CONSTRAINTS S
                         ,USER_CONS_COLUMNS L
                    WHERE LOWER(L.TABLE_NAME) = LOWER('` + tableName + `')
                      AND L.TABLE_NAME = S.TABLE_NAME
                      AND L.CONSTRAINT_NAME = S.CONSTRAINT_NAME
                      AND S.CONSTRAINT_TYPE = 'P') M
                ON M.TABLE_NAME = T.TABLE_NAME
               AND M.COLUMN_NAME = T.COLUMN_NAME
                  ,USER_COL_COMMENTS C
             WHERE LOWER(T.TABLE_NAME) = LOWER('` + tableName + `')
               AND C.TABLE_NAME = T.TABLE_NAME
               AND C.COLUMN_NAME = T.COLUMN_NAME
               AND T.HIDDEN_COLUMN = 'NO'
             ORDER BY T.COLUMN_ID`
}

func (t *goracleTemp) DecodeQuerySql(querySql string) string {
    if "initTable" == querySql {
        return `select table_name as name from user_tables
                union all
                select synonym_name||'≈'||table_owner||'.'||table_name as name from user_synonyms
                order by name`
    } else if strings.HasPrefix(querySql, "processShowColumn ") {
        tableName := querySql[len("processShowColumn "):]
        return t.DescribeTable(tableName)
    } else if strings.HasPrefix(querySql, "showCreateTable ") {
        tableName := querySql[len("showCreateTable "):]
        return `SELECT U.OBJECT_NAME, DBMS_METADATA.GET_DDL(U.OBJECT_TYPE, U.OBJECT_NAME)
                  FROM USER_OBJECTS U
                 WHERE U.OBJECT_TYPE IN ('TABLE', 'SYNONYM')
                   AND U.OBJECT_NAME = '` + tableName + `'`
    }
    return ""
}

func (t *goracleTemp) TableColumnsSql(dbName string) string {
    return `SELECT T.TABLE_NAME
                  ,T.COLUMN_NAME
                  ,C.COMMENTS AS "COLUMN_COMMENT"
                  ,(SELECT CASE WHEN T.COLUMN_NAME = M.COLUMN_NAME THEN 'PRI' ELSE '' END FROM DUAL) AS "COLUMN_KEY"
                  ,T.DATA_TYPE||DECODE (T.DATA_TYPE,
                                        'NUMBER', DECODE ('('
                                                          || NVL (TO_CHAR (DATA_PRECISION), '*')
                                                          || ','
                                                          || NVL (TO_CHAR (DATA_SCALE), '*')
                                                          || ')',
                                                          '(*,*)', NULL,
                                                          '(*,0)', '(38)',
                                                          '('
                                                          || NVL (TO_CHAR (DATA_PRECISION), '*')
                                                          || ','
                                                          || NVL (TO_CHAR (DATA_SCALE), '*')
                                                          || ')'),
                                        'FLOAT', '(' || DATA_PRECISION || ')',
                                        'DATE', NULL,
                                        'TIMESTAMP(6)', NULL,
                                        '(' || DATA_LENGTH || ')') AS "COLUMN_TYPE"
                  ,(SELECT CASE WHEN T.NULLABLE = 'N' THEN 'NO' ELSE 'YES' END FROM DUAL) AS "IS_NULLABLE"
                  ,T.DATA_DEFAULT AS "COLUMN_DEFAULT"
              FROM USER_TAB_COLS T
              LEFT JOIN (
                   SELECT L.TABLE_NAME
                         ,L.COLUMN_NAME
                     FROM USER_CONSTRAINTS S
                         ,USER_CONS_COLUMNS L
                    WHERE L.TABLE_NAME = S.TABLE_NAME
                      AND L.CONSTRAINT_NAME = S.CONSTRAINT_NAME
                      AND S.CONSTRAINT_TYPE = 'P') M
                ON M.TABLE_NAME = T.TABLE_NAME
               AND M.COLUMN_NAME = T.COLUMN_NAME
                  ,USER_COL_COMMENTS C
             WHERE C.TABLE_NAME = T.TABLE_NAME
               AND C.COLUMN_NAME = T.COLUMN_NAME
               AND T.HIDDEN_COLUMN = 'NO'
             ORDER BY T.TABLE_NAME, T.COLUMN_ID`
}

func (t *goracleTemp) TableCommentSql(dbName string) string {
    return `SELECT T.TABLE_NAME
                  ,T.COMMENTS AS "TABLE_COMMENT"
              FROM USER_TAB_COMMENTS T
             ORDER BY T.TABLE_NAME`
}

var goracleInstance = &goracleTemp{abstractSqlTemp{}}

// mysql

type mysqlTemp struct {
    abstractSqlTemp
}

func (t *mysqlTemp) SelectDb() string {
    return "SELECT DATABASE()"
}

func (t *mysqlTemp) SelectDbByTidResult(row []string) (string, string, string, error) {
    // user:pass@tcp(127.0.0.1:3306)/db?charset=utf8
    return "mysql", row[1] + ":" + row[2] +
        "@tcp(" + row[3] + ":" + row[4] + ")/" +
        row[5] + "?charset=utf8mb4,utf8&timeout=3s", row[5], nil
}

func (t *mysqlTemp) DescribeTable(tableName string) string {
    return "DESC " + tableName
}

func (t *mysqlTemp) DecodeQuerySql(querySql string) string {
    if "initTable" == querySql {
        return "show tables"
    } else if strings.HasPrefix(querySql, "processShowColumn ") {
        tableName := querySql[len("processShowColumn "):]
        return "show full columns from " + tableName
    } else if strings.HasPrefix(querySql, "showCreateTable ") {
        tableName := querySql[len("showCreateTable "):]
        return "show create table " + tableName
    }
    return ""
}

func (t *mysqlTemp) TableColumnsSql(dbName string) string {
    return `select TABLE_NAME, COLUMN_NAME, COLUMN_COMMENT, COLUMN_KEY, COLUMN_TYPE, IS_NULLABLE, COLUMN_DEFAULT
            from INFORMATION_SCHEMA.COLUMNS
            where TABLE_SCHEMA = '` + dbName + `' order by TABLE_NAME`
}

func (t *mysqlTemp) TableCommentSql(dbName string) string {
    return `select TABLE_NAME, TABLE_COMMENT from INFORMATION_SCHEMA.TABLES ` +
        `where TABLE_SCHEMA = '` + dbName + `'`
}

var mysqlInstance = &mysqlTemp{abstractSqlTemp{}}

// Sql of DriverName

func SqlOf(driverName string) SqlTemp {
    switch driverName {
    case "goracle":
        return goracleInstance
    case "mysql":
        return mysqlInstance
    default:
        panic("unsupported")
    }
}

// codec of Sql from Ajax

func IsCodedSql(querySql string) bool {
    return "initTable" == querySql ||
        strings.HasPrefix(querySql, "processShowColumn ") ||
        strings.HasPrefix(querySql, "showCreateTable ")
}
