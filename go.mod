module github.com/jansemmelink/rest

go 1.12

require (
	github.com/gorilla/context v1.1.1 // indirect
	github.com/gorilla/mux v1.7.3 // indirect
	github.com/gorilla/pat v1.0.1
	github.com/jansemmelink/items2 v0.0.0-20190731094523-396217f37cbc
	github.com/jansemmelink/log v0.3.0
)

replace github.com/jansemmelink/items2 => ../items2
