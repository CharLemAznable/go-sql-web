package main

import (
    "github.com/bingoohuang/gou/htt"
    "github.com/gorilla/mux"
    "net/http"
)

func serveFont(prefix string) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        path := prefix + mux.Vars(r)["extension"]
        htt.ServeFavicon(path, MustAsset, AssetInfo)(w, r)
    }
}
