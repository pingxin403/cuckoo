module github.com/pingxin403/cuckoo/libs/reconcile

go 1.21

require github.com/pingxin403/cuckoo/libs/hlc v0.0.0

require pgregory.net/rapid v1.2.0 // indirect

replace github.com/pingxin403/cuckoo/libs/hlc => ../hlc
