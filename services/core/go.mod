module github.com/aetherium/aetherium/services/core

go 1.25.3

require (
	github.com/aetherium/aetherium/libs/common v0.0.0
	github.com/aetherium/aetherium/libs/types v0.0.0
	github.com/google/uuid v1.6.0
)

require gopkg.in/yaml.v3 v3.0.1 // indirect

replace (
	github.com/aetherium/aetherium/libs/common => ../../libs/common
	github.com/aetherium/aetherium/libs/types => ../../libs/types
	github.com/aetherium/aetherium/services/gateway => ../../services/gateway
)
