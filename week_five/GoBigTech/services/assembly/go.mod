module github.com/bulbahal/GoBigTech/services/assembly

go 1.24.4

require (
	github.com/bulbahal/GoBigTech/platform v0.0.0
	github.com/google/uuid v1.6.0
	go.uber.org/zap v1.27.1
)

replace github.com/bulbahal/GoBigTech/platform => ../../platform
