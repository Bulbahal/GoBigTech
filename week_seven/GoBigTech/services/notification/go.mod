module github.com/bulbahal/GoBigTech/services/notification

go 1.24.4

require (
	github.com/bulbahal/GoBigTech/platform v0.0.0
	go.uber.org/zap v1.27.1
)

require (
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	github.com/segmentio/kafka-go v0.4.50 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/net v0.45.0 // indirect
	golang.org/x/text v0.30.0 // indirect
)

replace github.com/bulbahal/GoBigTech/platform => ../../platform
