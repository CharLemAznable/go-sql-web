package main

import (
    "database/sql"
    "encoding/json"
    "github.com/bingoohuang/gonet"
    "github.com/bingoohuang/sqlmore"
    "net/http"
    "strconv"
    "strings"
)

type UpdateResultRow struct {
    Ok      bool
    Message string
}

type UpdateResult struct {
    Ok         bool
    Message    string
    RowsResult []UpdateResultRow
}

func serveUpdate(w http.ResponseWriter, r *http.Request) {
    gonet.ContentTypeJSON(w)

    if !writeAuthOk(r) {
        http.Error(w, "write auth required", 405)
        return
    }

    sqls := strings.TrimSpace(r.FormValue("sqls"))
    tid := strings.TrimSpace(r.FormValue("tid"))

    driverName, dataSource, _, err := selectDb(tid)
    if err != nil {
        updateResult := UpdateResult{Ok: false, Message: err.Error()}
        _ = json.NewEncoder(w).Encode(updateResult)
        return
    }

    db, err := sql.Open(driverName, dataSource)
    if err != nil {
        updateResult := UpdateResult{Ok: false, Message: err.Error()}
        _ = json.NewEncoder(w).Encode(updateResult)
        return
    }
    defer db.Close()

    resultRows := make([]UpdateResultRow, 0)
    for _, s := range strings.Split(sqls, ";\n") {
        saveHistory(tid, s)
        sqlResult := sqlmore.ExecSQL(db, s, 0, "(null)")
        if sqlResult.Error != nil {
            resultRows = append(resultRows, UpdateResultRow{Ok: false, Message: "Error: " + sqlResult.Error.Error()})
        } else if sqlResult.RowsAffected == 1 {
            resultRows = append(resultRows, UpdateResultRow{Ok: true, Message: "1 rows affected!"})
        } else {
            resultRows = append(resultRows, UpdateResultRow{Ok: false, Message: strconv.FormatInt(sqlResult.RowsAffected, 10) + " rows affected!"})
        }
    }

    updateResult := UpdateResult{Ok: true, Message: "Ok", RowsResult: resultRows}
    _ = json.NewEncoder(w).Encode(updateResult)
}
