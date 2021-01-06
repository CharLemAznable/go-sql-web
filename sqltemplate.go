package main

import "C"
import (
    "strings"
)

type SqlTemp interface {
    SelectDb() string
    SelectDbByTidResult(row []string) (string, string, string, error)
    QualifyTable(tableName string) string
    DescribeTable(tableName string) string
    DecodeQuerySql(querySql string) string
    TableColumnsSql(dbName string) string
    TableCommentSql(dbName string) string
}

// goracle

type goracleTemp struct{}

func (t *goracleTemp) parseTableName(tableName string) (string, string) {
    pos := strings.Index(tableName, ".")
    if pos >= 0 {
        return tableName[:pos], tableName[pos+1:]
    }
    return "", tableName
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

func (t *goracleTemp) QualifyTable(tableName string) string {
    owner, objectName := t.parseTableName(tableName)
    if "" == owner {
        return `SELECT UPPER(USER||'.` + objectName + `') FROM DUAL`
    }
    return `SELECT UPPER('` + tableName + `') FROM DUAL`
}

func (t *goracleTemp) DescribeTable(tableName string) string {
    owner, objectName := t.parseTableName(tableName)
    ownerCond := `= UPPER('` + owner + `')`
    if "" == owner {
        ownerCond = `= USER`
    }
    return `SELECT
       T.COLUMN_NAME AS "FIELD"
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
                            'CLOB', NULL,
                            'BLOB', NULL,
                            '(' || DATA_LENGTH || ')') AS "TYPE"
      ,(SELECT CASE WHEN T.NULLABLE = 'N' THEN 'NO' ELSE 'YES' END FROM DUAL) AS "NULL"
      ,(SELECT CASE WHEN T.COLUMN_NAME = M.COLUMN_NAME THEN 'PRI' ELSE '' END FROM DUAL) AS "KEY"
      ,T.DATA_DEFAULT AS "DEFAULT"
      ,C.COMMENTS AS "Comment"
  FROM ALL_TAB_COLS T
  LEFT JOIN (
       SELECT L.TABLE_NAME
             ,L.COLUMN_NAME
         FROM ALL_CONSTRAINTS S
             ,ALL_CONS_COLUMNS L
        WHERE L.TABLE_NAME = S.TABLE_NAME
          AND L.CONSTRAINT_NAME = S.CONSTRAINT_NAME
          AND S.CONSTRAINT_TYPE = 'P') M
    ON M.TABLE_NAME = T.TABLE_NAME
   AND M.COLUMN_NAME = T.COLUMN_NAME
      ,ALL_COL_COMMENTS C
      ,(SELECT AO.OWNER||'.'||AO.OBJECT_NAME AS "FULL_NAME"
              ,CASE AO.OBJECT_TYPE
               WHEN 'TABLE' THEN AO.OWNER
               WHEN 'SYNONYM' THEN AN.TABLE_OWNER
               ELSE NULL END AS "OWNER"
              ,CASE AO.OBJECT_TYPE
               WHEN 'TABLE' THEN AO.OBJECT_NAME
               WHEN 'SYNONYM' THEN AN.TABLE_NAME
               ELSE NULL END AS "OBJECT_NAME"
          FROM ALL_OBJECTS AO
          LEFT JOIN ALL_USERS AU
            ON AO.OWNER = AU.USERNAME
          LEFT JOIN ALL_SYNONYMS AN
            ON AO.OWNER = AN.OWNER
           AND AO.OBJECT_NAME = AN.SYNONYM_NAME
           AND AO.OBJECT_TYPE = 'SYNONYM'
         WHERE AU.ORACLE_MAINTAINED = 'N'
           AND AO.OBJECT_TYPE IN ('TABLE', 'SYNONYM')
           AND UPPER(AO.OWNER) ` + ownerCond + `
           AND UPPER(AO.OBJECT_NAME) = UPPER('` + objectName + `')) O
 WHERE T.OWNER = O.OWNER
   AND T.TABLE_NAME = O.OBJECT_NAME
   AND C.TABLE_NAME = T.TABLE_NAME
   AND C.COLUMN_NAME = T.COLUMN_NAME
   AND T.HIDDEN_COLUMN = 'NO'
 ORDER BY T.COLUMN_ID`
}

func (t *goracleTemp) DecodeQuerySql(querySql string) string {
    if "initTable" == querySql {
        return `
SELECT AO.OWNER||'.'||AO.OBJECT_NAME
  FROM ALL_OBJECTS AO
  LEFT JOIN ALL_USERS AU
    ON AO.OWNER = AU.USERNAME
  LEFT JOIN ALL_SYNONYMS AN
    ON AO.OWNER = AN.OWNER
   AND AO.OBJECT_NAME = AN.SYNONYM_NAME
   AND AO.OBJECT_TYPE = 'SYNONYM'
 WHERE AU.ORACLE_MAINTAINED = 'N'
   AND AO.OBJECT_TYPE IN ('TABLE', 'SYNONYM')
 ORDER BY DECODE(AO.OWNER, USER, 1, 2), AO.OWNER, AO.OBJECT_NAME`
    } else if strings.HasPrefix(querySql, "processShowColumn ") {
        tableName := querySql[len("processShowColumn "):]
        return t.DescribeTable(tableName)
    } else if strings.HasPrefix(querySql, "showCreateTable ") {
        tableName := querySql[len("showCreateTable "):]
        owner, objectName := t.parseTableName(tableName)
        ownerCond := `= UPPER('` + owner + `')`
        if "" == owner {
            ownerCond = `= USER`
        }
        return `
SELECT AO.OWNER||'.'||AO.OBJECT_NAME
      ,DBMS_METADATA.GET_DDL(AO.OBJECT_TYPE, AO.OBJECT_NAME, AO.OWNER)
  FROM ALL_OBJECTS AO
 WHERE AO.OBJECT_TYPE IN ('TABLE', 'SYNONYM')
   AND UPPER(AO.OWNER) ` + ownerCond + `
   AND UPPER(AO.OBJECT_NAME) = UPPER('` + objectName + `')`
    }
    return ""
}

func (t *goracleTemp) TableColumnsSql(dbName string) string {
    return `
SELECT O.FULL_NAME AS "TABLE_NAME"
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
                            'CLOB', NULL,
                            'BLOB', NULL,
                            '(' || DATA_LENGTH || ')') AS "COLUMN_TYPE"
      ,(SELECT CASE WHEN T.NULLABLE = 'N' THEN 'NO' ELSE 'YES' END FROM DUAL) AS "IS_NULLABLE"
      ,T.DATA_DEFAULT AS "COLUMN_DEFAULT"
  FROM ALL_TAB_COLS T
  LEFT JOIN (
       SELECT L.TABLE_NAME
             ,L.COLUMN_NAME
         FROM ALL_CONSTRAINTS S
             ,ALL_CONS_COLUMNS L
        WHERE L.TABLE_NAME = S.TABLE_NAME
          AND L.CONSTRAINT_NAME = S.CONSTRAINT_NAME
          AND S.CONSTRAINT_TYPE = 'P') M
    ON M.TABLE_NAME = T.TABLE_NAME
   AND M.COLUMN_NAME = T.COLUMN_NAME
      ,ALL_COL_COMMENTS C
      ,(SELECT AO.OWNER||'.'||AO.OBJECT_NAME AS "FULL_NAME"
              ,CASE AO.OBJECT_TYPE
               WHEN 'TABLE' THEN AO.OWNER
               WHEN 'SYNONYM' THEN AN.TABLE_OWNER
               ELSE NULL END AS "OWNER"
              ,CASE AO.OBJECT_TYPE
               WHEN 'TABLE' THEN AO.OBJECT_NAME
               WHEN 'SYNONYM' THEN AN.TABLE_NAME
               ELSE NULL END AS "OBJECT_NAME"
          FROM ALL_OBJECTS AO
          LEFT JOIN ALL_USERS AU
            ON AO.OWNER = AU.USERNAME
          LEFT JOIN ALL_SYNONYMS AN
            ON AO.OWNER = AN.OWNER
           AND AO.OBJECT_NAME = AN.SYNONYM_NAME
           AND AO.OBJECT_TYPE = 'SYNONYM'
         WHERE AU.ORACLE_MAINTAINED = 'N'
           AND AO.OBJECT_TYPE IN ('TABLE', 'SYNONYM')) O
         WHERE T.OWNER = O.OWNER
   AND T.TABLE_NAME = O.OBJECT_NAME
   AND C.TABLE_NAME = T.TABLE_NAME
   AND C.COLUMN_NAME = T.COLUMN_NAME
   AND T.HIDDEN_COLUMN = 'NO'
 ORDER BY O.FULL_NAME, T.COLUMN_ID`
}

func (t *goracleTemp) TableCommentSql(dbName string) string {
    return `
SELECT O.FULL_NAME AS "TABLE_NAME"
      ,T.COMMENTS AS "TABLE_COMMENT"
  FROM ALL_TAB_COMMENTS T
      ,(SELECT AO.OWNER||'.'||AO.OBJECT_NAME AS "FULL_NAME"
              ,CASE AO.OBJECT_TYPE
               WHEN 'TABLE' THEN AO.OWNER
               WHEN 'SYNONYM' THEN AN.TABLE_OWNER
               ELSE NULL END AS "OWNER"
              ,CASE AO.OBJECT_TYPE
               WHEN 'TABLE' THEN AO.OBJECT_NAME
               WHEN 'SYNONYM' THEN AN.TABLE_NAME
               ELSE NULL END AS "OBJECT_NAME"
          FROM ALL_OBJECTS AO
          LEFT JOIN ALL_USERS AU
            ON AO.OWNER = AU.USERNAME
          LEFT JOIN ALL_SYNONYMS AN
            ON AO.OWNER = AN.OWNER
           AND AO.OBJECT_NAME = AN.SYNONYM_NAME
           AND AO.OBJECT_TYPE = 'SYNONYM'
         WHERE AU.ORACLE_MAINTAINED = 'N'
           AND AO.OBJECT_TYPE IN ('TABLE', 'SYNONYM')) O
 WHERE T.OWNER = O.OWNER
   AND T.TABLE_NAME = O.OBJECT_NAME
 ORDER BY O.FULL_NAME`
}

var goracleInstance = &goracleTemp{}

// mysql

type mysqlTemp struct{}

func (t *mysqlTemp) SelectDb() string {
    return "SELECT DATABASE()"
}

func (t *mysqlTemp) SelectDbByTidResult(row []string) (string, string, string, error) {
    // user:pass@tcp(127.0.0.1:3306)/db?charset=utf8
    return "mysql", row[1] + ":" + row[2] +
        "@tcp(" + row[3] + ":" + row[4] + ")/" +
        row[5] + "?charset=utf8mb4,utf8&timeout=3s", row[5], nil
}

func (t *mysqlTemp) QualifyTable(tableName string) string {
    return `SELECT LOWER('` + tableName + `')`
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

var mysqlInstance = &mysqlTemp{}

// mssql

type mssqlTemp struct {}

func (t *mssqlTemp) SelectDb() string {
    return "SELECT NAME FROM SYSDATABASES"
}

func (t *mssqlTemp) SelectDbByTidResult(row []string) (string, string, string, error) {
    // sqlserver://user:pass@host:port/instance
    return "mssql", "sqlserver://" + row[1] + ":" + row[2] +
        "@" + row[3] + ":" + row[4] +
        "?database=" + row[5] + "&connection+timeout=30", row[5], nil
}

func (t *mssqlTemp) QualifyTable(tableName string) string {
    return "\"" + tableName + "\""
}

func (t *mssqlTemp) DescribeTable(tableName string) string {
    return `
SELECT COLUMN_NAME
      ,TABLE_NAME
      ,'' AS "TYPE"
      ,'PRI' AS "KEY"
  FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
 WHERE TABLE_NAME = '` + tableName + `'`
}

func (t *mssqlTemp) DecodeQuerySql(querySql string) string {
    if "initTable" == querySql {
        return "SELECT NAME FROM SYS.TABLES ORDER BY NAME"
    } else if strings.HasPrefix(querySql, "processShowColumn ") {
        tableName := querySql[len("processShowColumn "):]
        return "SELECT * FROM SYSCOLUMNS WHERE ID = OBJECT_ID('" + tableName + "')"
    } else if strings.HasPrefix(querySql, "showCreateTable ") {
        tableName := querySql[len("showCreateTable "):]
        return "SELECT '" + tableName + "', '-- Show Create Table Not Supported'" // TODO
    }
    return ""
}

func (t *mssqlTemp) TableColumnsSql(dbName string) string {
    return `
SELECT OBJECT_NAME(A.ID)               AS "TABLE_NAME"
      ,A.NAME                          AS "COLUMN_NAME"
      ,ISNULL(G.[VALUE], '')           AS "COLUMN_COMMENT"
      ,CASE
       WHEN EXISTS(SELECT 1 FROM SYSOBJECTS WHERE XTYPE = 'PK' AND PARENT_OBJ = A.ID AND NAME IN (
                   SELECT NAME FROM SYSINDEXES WHERE INDID IN (
                   SELECT INDID FROM SYSINDEXKEYS WHERE ID = A.ID AND COLID = A.COLID))) THEN 'PRI'
       ELSE '' END                     AS "COLUMN_KEY"
      ,B.NAME                          AS "COLUMN_TYPE"
      ,CASE 
       WHEN A.ISNULLABLE = 1 THEN 'YES'
       ELSE 'NO' END                   AS "IS_NULLABLE"
      ,ISNULL(E.TEXT, '')              AS "COLUMN_DEFAULT"
  FROM SYSCOLUMNS                      A
  LEFT JOIN SYSTYPES                   B
    ON A.XUSERTYPE                     = B.XUSERTYPE
 INNER JOIN SYSOBJECTS                 D
    ON A.ID                            = D.ID
   AND D.XTYPE                         = 'U'
   AND D.NAME                          <> 'DTPROPERTIES'
  LEFT JOIN SYSCOMMENTS                E
    ON A.CDEFAULT                      = E.ID
  LEFT JOIN SYS.EXTENDED_PROPERTIES    G 
    ON A.ID                            = G.MAJOR_ID
   AND A.COLID                         = G.MINOR_ID  
  LEFT JOIN SYS.EXTENDED_PROPERTIES    F
    ON D.ID                            = F.MAJOR_ID
   AND F.MINOR_ID                      = 0
 ORDER BY D.NAME`
}

func (t *mssqlTemp) TableCommentSql(dbName string) string {
    return `
SELECT DISTINCT
       D.NAME                       AS "TABLE_NAME"
      ,ISNULL(F.VALUE, '')          AS "TABLE_COMMENT"
  FROM SYSCOLUMNS                   A
  LEFT JOIN SYSTYPES                B
    ON A.XUSERTYPE                  = B.XUSERTYPE
 INNER JOIN SYSOBJECTS              D
    ON A.ID                         = D.ID 
   AND D.XTYPE                      = 'U'
   AND D.NAME                       <> 'DTPROPERTIES'
  LEFT JOIN SYSCOMMENTS             E
    ON A.CDEFAULT                   = E.ID
  LEFT JOIN SYS.EXTENDED_PROPERTIES G
    ON A.ID                         = G.MAJOR_ID
   AND A.COLID                      = G.MINOR_ID
  LEFT JOIN SYS.EXTENDED_PROPERTIES F
    ON D.ID                         = F.MAJOR_ID 
   AND F.MINOR_ID                   = 0`
}

var mssqlInstance = &mssqlTemp{}

// Sql of DriverName

func SqlOf(driverName string) SqlTemp {
    switch driverName {
    case "goracle":
        return goracleInstance
    case "mysql":
        return mysqlInstance
    case "mssql":
        return mssqlInstance
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
