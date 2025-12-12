module github.com/aetherium/aetherium/services/k8s-manager

go 1.25.3

replace (
	github.com/aetherium/aetherium/libs/common => ../../libs/common
	github.com/aetherium/aetherium/libs/types => ../../libs/types
	github.com/aetherium/aetherium/services/core => ../core
)
