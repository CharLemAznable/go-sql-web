package main

import (
    "encoding/json"
    "errors"
    "github.com/bingoohuang/gonet"
    "github.com/mgutz/str"
    "net/http"
    "strconv"
    "strings"
)

type Merchant struct {
    MerchantName string
    MerchantId   string
    MerchantCode string
    HomeArea     string
    Classifier   string
}

func serveSearchDb(w http.ResponseWriter, req *http.Request) {
    gonet.ContentTypeJSON(w)
    searchKey := strings.TrimSpace(req.FormValue("searchKey"))
    byTenant := strings.TrimSpace(req.FormValue("byTenant"))
    if searchKey == "" {
        http.Error(w, "searchKey required", 405)
        return
    }

    if searchKey == "trr" || !appConfig.MultiTenants {
        var searchResult [1]Merchant
        searchResult[0] = Merchant{
            MerchantName: "trr",
            MerchantId:   "trr",
            MerchantCode: "trr",
            HomeArea:     "south-center",
            Classifier:   "trr"}
        _ = json.NewEncoder(w).Encode(searchResult)
        return
    }

    values := map[string]interface{}{"searchKey": searchKey}
    searchSql := ""
    if byTenant == "true" {
        searchSql = str.Template(appConfig.SearchDbMerchantByTenantSQL, values)
    } else {
        searchSql = str.Template(appConfig.SearchDbMerchantNotByTenantSQL, values)
    }
    _, data, _, _, err, _ := executeQuery(searchSql, appConfig.DriverName, appConfig.DataSource, 0)
    if err != nil {
        http.Error(w, err.Error(), 405)
        return
    }

    searchResult := make([]Merchant, 0, len(data)+1)

    if len(data) == 0 {
        searchResult = append(searchResult,
            Merchant{MerchantName: "trr", MerchantId: "trr", MerchantCode: "trr", HomeArea: "south-center", Classifier: "trr"})
    } else {
        for _, v := range data {
            tid := v[2]
            if tid != "trr" {
                searchResult = append(searchResult,
                    Merchant{MerchantName: v[1], MerchantId: tid, MerchantCode: v[3], HomeArea: v[4], Classifier: v[5]})
            }
        }
    }
    _ = json.NewEncoder(w).Encode(searchResult)

}

type MerchantDb struct {
    MerchantId string
    Username   string
    Password   string
    Host       string
    Port       string
    Database   string
}

func searchMerchantDb(tid string, dn, ds string) (*MerchantDb, error) {
    sql := str.Template(appConfig.SearchMerchantDbByTidSQL, map[string]interface{}{"tid": tid})

    _, data, _, _, err, _ := executeQuery(sql, dn, ds, 1)
    if err != nil {
        return nil, err
    }

    if len(data) != 1 {
        return nil, errors.New("none or more than one found for tid:" + tid)
    }
    v := data[0]

    return &MerchantDb{MerchantId: v[1], Username: v[2], Password: v[3], Host: v[4], Port: v[5], Database: v[6]}, nil
}

func searchMerchant(tid string) (*Merchant, error) {
    if tid == "trr" {
        return &Merchant{MerchantName: tid, MerchantId: tid, MerchantCode: tid, HomeArea: appConfig.TrrHomeArea, Classifier: tid}, nil
    }

    sql := str.Template(appConfig.SearchMerchantByTidSQL, map[string]interface{}{"tid": tid})

    return searchMerchantBySql(sql, 1)
}

func searchMerchantByTcode(tcode string) (*Merchant, error) {
    sql := str.Template(appConfig.SearchMerchantByTcodeSQL, map[string]interface{}{"tcode": tcode})

    return searchMerchantBySql(sql, 1)
}

func searchMerchantBySql(searchSql string, maxRows int) (*Merchant, error) {
    _, data, _, _, err, _ := executeQuery(searchSql, appConfig.DriverName, appConfig.DataSource, maxRows)
    if err != nil {
        return nil, err
    }

    if len(data) != 1 {
        return nil, errors.New("merchant query result " + strconv.Itoa(len(data)) + " other than 1")
    }

    v := data[0]

    return &Merchant{MerchantName: v[1], MerchantId: v[2], MerchantCode: v[3], HomeArea: v[4], Classifier: v[5]}, nil
}
