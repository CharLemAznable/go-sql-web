package main

import (
    "flag"
    "github.com/BurntSushi/toml"
    "github.com/CharLemAznable/amber"
    "github.com/bingoohuang/gou/htt"
    "log"
    "strings"
)

type YogaProxy struct {
    Proxy string
}

type AppConfig struct {
    ContextPath   string
    ListenPort    int
    MaxQueryRows  int
    DriverName    string // goracle/mysql/pq/go-sqlite3
    DataSource    string
    DefaultTenant string
    TrrHomeArea   string

    DevMode       bool // to disable css/js minify
    AuthBasic     bool
    AuthBasicUser string
    AuthBasicPass string
    MultiTenants  bool
    ImportDb      bool

    YogaProxy map[string]YogaProxy

    EncryptKey  string
    CookieName  string
    RedirectUri string
    LocalUrl    string
    ForceLogin  bool

    AmberLoginEnabled  bool
    AmberAppId         string
    AmberEncryptKey    string
    AmberCookieName    string
    AmberAmberLoginUrl string
    AmberLocalUrl      string
    AmberForceLogin    bool

    WriteAuthUserNames []string // UserNames which has write auth

    SearchDbMerchantByTenantSQL    string
    SearchDbMerchantNotByTenantSQL string
    SearchMerchantByTidSQL         string
    SearchMerchantByTcodeSQL       string
    SearchMerchantDbByTidSQL       string
    SearchDbByTidSQL               string

    MultipleTenantsExecConfig map[string][]string
}

var configFile string
var appConfig AppConfig

var authParam htt.MustAuthParam

func init() {
    flag.StringVar(&configFile, "configFile", "appConfig.toml", "config file path")
    flag.StringVar(&configFile, "c", "appConfig.toml", "config file path(shorthand)")

    flag.Parse()
    if _, err := toml.DecodeFile(configFile, &appConfig); err != nil {
        log.Panic("config file decode error", err.Error())
    }

    if appConfig.ContextPath != "" && strings.Index(appConfig.ContextPath, "/") < 0 {
        appConfig.ContextPath = "/" + appConfig.ContextPath
    }

    authParam = htt.MustAuthParam{
        EncryptKey:  appConfig.EncryptKey,
        CookieName:  appConfig.CookieName,
        RedirectURI: appConfig.RedirectUri,
        LocalURL:    appConfig.LocalUrl,
        ForceLogin:  appConfig.ForceLogin,
    }
    htt.PrepareMustAuthFlag(&authParam)

    if appConfig.AmberLoginEnabled {
        amber.ConfigInstance = amber.NewConfig(
            amber.WithAppId(appConfig.AmberAppId),
            amber.WithEncryptKey(appConfig.AmberEncryptKey),
            amber.WithCookieName(appConfig.AmberCookieName),
            amber.WithAmberLoginUrl(appConfig.AmberAmberLoginUrl),
            amber.WithLocalUrl(appConfig.AmberLocalUrl),
            amber.WithForceLogin(appConfig.AmberForceLogin),
        )
    }

    if 0 == len(appConfig.SearchDbMerchantByTenantSQL) {
        appConfig.SearchDbMerchantByTenantSQL = "SELECT MERCHANT_NAME, MERCHANT_ID, MERCHANT_CODE, HOME_AREA, CLASSIFIER " +
            "FROM TR_F_MERCHANT WHERE MERCHANT_ID = '{{searchKey}}' OR MERCHANT_CODE = '{{searchKey}}'"
    }
    if 0 == len(appConfig.SearchDbMerchantNotByTenantSQL) {
        appConfig.SearchDbMerchantNotByTenantSQL = "SELECT MERCHANT_NAME, MERCHANT_ID, MERCHANT_CODE, HOME_AREA, CLASSIFIER " +
            "FROM TR_F_MERCHANT WHERE MERCHANT_ID = '{{searchKey}}' OR MERCHANT_CODE = '{{searchKey}}' " +
            "OR MERCHANT_NAME LIKE '%{{searchKey}}%'"
    }
    if 0 == len(appConfig.SearchMerchantByTidSQL) {
        appConfig.SearchMerchantByTidSQL = "SELECT MERCHANT_NAME, MERCHANT_ID, MERCHANT_CODE, HOME_AREA, CLASSIFIER " +
            "FROM TR_F_MERCHANT WHERE MERCHANT_ID = '{{tid}}'"
    }
    if 0 == len(appConfig.SearchMerchantByTcodeSQL) {
        appConfig.SearchMerchantByTcodeSQL = "SELECT MERCHANT_NAME, MERCHANT_ID, MERCHANT_CODE, HOME_AREA, CLASSIFIER " +
            "FROM TR_F_MERCHANT WHERE MERCHANT_CODE = '{{tcode}}'"
    }
    if 0 == len(appConfig.SearchMerchantDbByTidSQL) {
        appConfig.SearchMerchantDbByTidSQL = "SELECT MERCHANT_ID, DB_USERNAME, DB_PASSWORD, PROXY_IP, PROXY_PORT, DB_NAME " +
            "FROM TR_F_DB WHERE MERCHANT_ID = '{{tid}}'"
    }
    if 0 == len(appConfig.SearchDbByTidSQL) {
        appConfig.SearchDbByTidSQL = "SELECT DB_USERNAME, DB_PASSWORD, PROXY_IP, PROXY_PORT, DB_NAME " +
            "FROM TR_F_DB WHERE MERCHANT_ID = '{{tid}}'"
    }

    if 0 == len(appConfig.MultipleTenantsExecConfig) {
        appConfig.MultipleTenantsExecConfig = map[string][]string{
            "TR_F_MERCHANT": {"MERCHANT_ID", "MERCHANT_NAME", "MERCHANT_CODE"},
            "TR_F_DB":       {"MERCHANT_ID", "DB_NAME", "DB_USERNAME"},
        }
    }
}
