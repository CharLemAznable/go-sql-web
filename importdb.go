package main

import (
    "encoding/json"
    "github.com/bingoohuang/gonet"
    "io"
    "mime/multipart"
    "net/http"
    "os"
    "strings"
    "unicode"
)

type ImportResult struct {
    Success bool   `json:"success"`
    Result  string `json:"result"`
}

func importDatabase(w http.ResponseWriter, r *http.Request) {
    gonet.ContentTypeJSON(w)

    if !writeAuthOk(r) {
        http.Error(w, "write auth required", 405)
        return
    }

    if !CommandExist("mysql") {
        http.Error(w, "mysql client not well installed", 405)
        return
    }

    tcode, sqlFileName, err := ParseUploadedFile(r, w)
    if err != nil {
        http.Error(w, err.Error(), 405)
        return
    }

    t, err := searchMerchantByTcode(tcode)
    if err != nil {
        http.Error(w, err.Error(), 405)
        return
    }

    d, err := searchMerchantDb(t.MerchantId, appConfig.DriverName, appConfig.DataSource)
    if err != nil {
        http.Error(w, err.Error(), 405)
        return
    }

    mysqlImport := "mysql -h" + d.Host + " -P" + d.Port + " -u" + d.Username + " -p" + d.Password + " " + d.Database + "<" + sqlFileName
    stdout, err := ExecuteBash(mysqlImport)
    ignoredErr := "mysql: [Warning] Using a password on the command line interface can be insecure."
    if err != nil {
        errMsg := strings.TrimFunc(err.Error(), func(r rune) bool {
            return unicode.IsSpace(r) || !unicode.IsPrint(r)
        })
        if errMsg != ignoredErr {
            http.Error(w, errMsg, 405)
            return
        }
    }

    json.NewEncoder(w).Encode(&ImportResult{
        Success: true,
        Result:  stdout,
    })
}

func ParseUploadedFile(r *http.Request, w http.ResponseWriter) (string, string, error) {
    r.ParseMultipartForm(32 << 20)
    file, _, err := r.FormFile("file")
    if err != nil {
        return "", "", err
    }
    defer file.Close()

    tcode := strings.TrimSpace(r.FormValue("tcode"))
    fileName, err := WriteExportedSqlFile(tcode, w, file)
    return tcode, fileName, err
}

func WriteExportedSqlFile(tcode string, w http.ResponseWriter, file multipart.File) (string, error) {
    _ = os.MkdirAll("./exported", os.ModePerm)
    exportedSqlFile := "./exported/" + tcode + "." + TimeNow() + ".sql"
    f, err := os.OpenFile(exportedSqlFile, os.O_WRONLY|os.O_CREATE, 0666)
    if err != nil {
        return exportedSqlFile, err
    }
    defer f.Close()

    io.Copy(f, file)
    return exportedSqlFile, nil
}
