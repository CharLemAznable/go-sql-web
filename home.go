package main

import (
	"github.com/CharLemAznable/amber"
	"github.com/bingoohuang/gonet"
	"github.com/bingoohuang/gou/htt"
	"net/http"
	"strconv"
	"strings"
)

func loginedUserName(r *http.Request) string {
	cookieValue := r.Context().Value("CookieValue")
	if nil != cookieValue {
		cookie := cookieValue.(*htt.CookieValueImpl)
		return cookie.Name
	}

	cookieValue = r.Context().Value(amber.CookieValueContextKey)
	if nil != cookieValue {
		cookie := cookieValue.(*amber.CookieValue)
		return cookie.Username
	}

	return ""
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	gonet.ContentTypeHTML(w)
	loginedHtml := ""

	cookieValue := r.Context().Value("CookieValue")
	if nil != cookieValue {
		cookie := cookieValue.(*htt.CookieValueImpl)
		loginedHtml = `<span id="loginSpan"><img class="loginAvatar" src="` + cookie.Avatar +
			`"/><span class="loginName">` + cookie.Name + `</span></span>`
	}

	cookieValue = r.Context().Value(amber.CookieValueContextKey)
	if nil != cookieValue {
		cookie := cookieValue.(*amber.CookieValue)
		loginedHtml = `<span id="loginSpan"><span class="loginName">` + cookie.Username + `</span></span>`
	}

	indexHtml := string(MustAsset("index.html"))
	indexHtml = strings.Replace(indexHtml, "<LOGIN/>", loginedHtml, 1)

	html := htt.MinifyHTML(indexHtml, appConfig.DevMode)

	mergeCss := htt.MergeCSS(MustAsset, FilterAssetNames(AssetNames, ".css"))
	css := htt.MinifyCSS(mergeCss, appConfig.DevMode)
	mergeScripts := htt.MergeJs(MustAsset, FilterAssetNames(AssetNames, ".js"))
	js := htt.MinifyJs(mergeScripts, appConfig.DevMode)
	html = strings.Replace(html, "/*.CSS*/", css, 1)
	html = strings.Replace(html, "/*.SCRIPT*/", js, 1)
	html = strings.Replace(html, "${contextPath}", appConfig.ContextPath, -1)
	html = strings.Replace(html, "${multiTenants}", strconv.FormatBool(appConfig.MultiTenants), -1)
	html = strings.Replace(html, "${defaultTenant}", appConfig.DefaultTenant, -1)

	w.Write([]byte(html))
}

func FilterAssetNames(assetNames []string, suffix string) []string {
	filtered := make([]string, 0)
	for _, assetName := range assetNames {
		if !strings.HasPrefix(assetName, "static/") && strings.HasSuffix(assetName, suffix) {
			filtered = append(filtered, assetName)
		}
	}

	return filtered
}
