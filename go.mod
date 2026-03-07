module github.com/cuckoo-org/cuckoo

go 1.21

require (
	github.com/cuckoo-org/cuckoo/examples/mvp/queue v0.0.0-00010101000000-000000000000
	github.com/cuckoo-org/cuckoo/examples/mvp/storage v0.0.0-00010101000000-000000000000
	github.com/cuckoo-org/cuckoo/libs/hlc v0.0.0-00010101000000-000000000000
	pgregory.net/rapid v1.2.0
)

require github.com/mattn/go-sqlite3 v1.14.18 // indirect

replace github.com/cuckoo-org/cuckoo/examples/multi-region/health => ./examples/multi-region/health

replace github.com/cuckoo-org/cuckoo/libs/connpool => ./libs/connpool

replace github.com/cuckoo-org/cuckoo/libs/hlc => ./libs/hlc

replace github.com/cuckoo-org/cuckoo/examples/mvp/storage => ./examples/mvp/storage

replace github.com/cuckoo-org/cuckoo/examples/mvp/queue => ./examples/mvp/queue
