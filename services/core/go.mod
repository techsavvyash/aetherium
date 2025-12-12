module github.com/aetherium/aetherium/services/core

go 1.25

require (
	github.com/creack/pty v1.1.24
	github.com/firecracker-microvm/firecracker-go-sdk v1.0.0
	github.com/go-chi/chi/v5 v5.2.3
	github.com/golang-migrate/migrate/v4 v4.19.0
	github.com/google/uuid v1.6.0
	github.com/hibiken/asynq v0.25.1
	github.com/jmoiron/sqlx v1.4.0
	github.com/lib/pq v1.10.9
	github.com/mdlayher/vsock v1.2.1
	github.com/redis/go-redis/v9 v9.7.0
	github.com/stretchr/testify v1.11.1
	gopkg.in/yaml.v3 v3.0.1
	github.com/aetherium/aetherium/libs/common v0.0.0
	github.com/aetherium/aetherium/libs/types v0.0.0
)

replace (
	github.com/aetherium/aetherium/libs/common => ../../libs/common
	github.com/aetherium/aetherium/libs/types => ../../libs/types
)
