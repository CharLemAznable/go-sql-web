module github.com/CharLemAznable/go-sql-web

go 1.12

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/CharLemAznable/amber v0.3.0
	github.com/bingoohuang/gonet v0.0.0-20190729063044-0a2a8ec96e17
	github.com/bingoohuang/gou v0.0.0-20190724062522-59c35e658334
	github.com/bingoohuang/sqlmore v0.0.0-20190711152446-8687de30af5c
	github.com/bingoohuang/statiq v0.2.1
	github.com/denisenkom/go-mssqldb v0.9.0
	github.com/go-sql-driver/mysql v1.4.1
	github.com/google/go-cmp v0.3.0 // indirect
	github.com/gorilla/mux v1.6.2
	github.com/kr/pretty v0.1.0 // indirect
	github.com/lib/pq v1.1.1
	github.com/mattn/go-sqlite3 v1.10.0
	github.com/mgutz/str v1.2.0
	github.com/xwb1989/sqlparser v0.0.0-20180606152119-120387863bf2
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad // indirect
	golang.org/x/sync v0.0.0-20190423024810-112230192c58 // indirect
	gopkg.in/goracle.v2 v2.16.3
)

replace (
	github.com/tdewolff/parse => github.com/tdewolff/parse v0.0.0-20181024085210-fced451e0bed
	golang.org/x/sys => github.com/golang/sys v0.0.0-20190606203320-7fc4e5ec1444
)
