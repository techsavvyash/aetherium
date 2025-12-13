module github.com/aetherium/aetherium/libs/common

go 1.25.3

require (
	github.com/aetherium/aetherium/libs/types v0.0.0
	github.com/aetherium/aetherium/services/core v0.0.0
	github.com/aetherium/aetherium/services/gateway v0.0.0
	github.com/redis/go-redis/v9 v9.7.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/google/uuid v1.6.0 // indirect
)

replace (
	github.com/aetherium/aetherium/libs/types => ../../libs/types
	github.com/aetherium/aetherium/services/core => ../../services/core
	github.com/aetherium/aetherium/services/gateway => ../../services/gateway
)
