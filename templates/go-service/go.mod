module {{MODULE_PATH}}

go 1.21

require (
	github.com/google/uuid v1.6.0
	github.com/pingxin403/cuckoo/libs/observability v0.0.0
	google.golang.org/grpc v1.70.0
	google.golang.org/protobuf v1.36.4
)

replace github.com/pingxin403/cuckoo/libs/observability => ../../libs/observability
