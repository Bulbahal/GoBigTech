module github.com/bulbahal/GoBigTech/services/order

go 1.24.4

require (
	github.com/bulbahal/GoBigTech/services/inventory v0.0.0-00010101000000-000000000000
	github.com/bulbahal/GoBigTech/services/payment v0.0.0-00010101000000-000000000000
	github.com/go-chi/chi/v5 v5.2.3
	github.com/oapi-codegen/runtime v1.1.2
	google.golang.org/grpc v1.76.0
)

require (
	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/text v0.27.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250804133106-a7a43d27e69b // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)

replace github.com/bulbahal/GoBigTech/services/inventory => ../inventory

replace github.com/bulbahal/GoBigTech/services/payment => ../payment
