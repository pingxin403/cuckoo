module github.com/cuckoo-org/cuckoo/health

go 1.21

require (
	github.com/cuckoo-org/cuckoo/queue v0.0.0-00010101000000-000000000000
	github.com/cuckoo-org/cuckoo/storage v0.0.0-00010101000000-000000000000
	github.com/mattn/go-sqlite3 v1.14.18
)

replace github.com/cuckoo-org/cuckoo/queue => ../queue

replace github.com/cuckoo-org/cuckoo/storage => ../storage
