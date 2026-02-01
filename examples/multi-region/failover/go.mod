module github.com/cuckoo-org/cuckoo/failover

go 1.21

require (
	github.com/cuckoo-org/cuckoo/arbiter v0.0.0
	github.com/cuckoo-org/cuckoo/health v0.0.0
)

replace github.com/cuckoo-org/cuckoo/arbiter => ../arbiter
replace github.com/cuckoo-org/cuckoo/health => ../health