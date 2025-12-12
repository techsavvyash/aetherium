module github.com/aetherium/aetherium/services/k8s-manager

go 1.25

require (
	k8s.io/client-go v0.29.0
	k8s.io/api v0.29.0
	k8s.io/apimachinery v0.29.0
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
