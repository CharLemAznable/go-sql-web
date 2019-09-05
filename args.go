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
	AmberAppID         string
	AmberEncryptKey    string
	AmberCookieName    string
	AmberAmberLoginURL string
	AmberLocalURL      string
	AmberForceLogin    bool

	WriteAuthUserNames []string // UserNames which has write auth
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
			amber.WithAppId(appConfig.AmberAppID),
			amber.WithEncryptKey(appConfig.AmberEncryptKey),
			amber.WithCookieName(appConfig.AmberCookieName),
			amber.WithAmberLoginUrl(appConfig.AmberAmberLoginURL),
			amber.WithLocalUrl(appConfig.AmberLocalURL),
			amber.WithForceLogin(appConfig.AmberForceLogin),
		)
	}
}
