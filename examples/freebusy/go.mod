module github.com/oh-tarnished/generateql/examples/freebusy

go 1.26.4

require (
	github.com/google/uuid v1.6.0
	github.com/the-protobuf-project/runtime-go/network v0.0.0-20260707151848-75a55c595654
)

require (
	github.com/coder/websocket v1.8.14 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/hasura/go-graphql-client v0.16.0 // indirect
)

// The runtime lives in the single generateql module one directory up.
replace github.com/oh-tarnished/generateql => ../..
