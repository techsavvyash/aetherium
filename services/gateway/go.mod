module github.com/aetherium/aetherium/services/gateway

go 1.25

require (
	github.com/go-chi/chi/v5 v5.2.3
	github.com/go-chi/cors v1.2.2
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/redis/go-redis/v9 v9.7.0
	gopkg.in/yaml.v3 v3.0.1
	github.com/aetherium/aetherium/libs/common v0.0.0
	github.com/aetherium/aetherium/libs/types v0.0.0
	github.com/aetherium/aetherium/services/core v0.0.0
)

replace (
	github.com/aetherium/aetherium/libs/common => ../../libs/common
	github.com/aetherium/aetherium/libs/types => ../../libs/types
	github.com/aetherium/aetherium/services/core => ../core
)
