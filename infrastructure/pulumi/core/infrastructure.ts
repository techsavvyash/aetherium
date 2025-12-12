import * as pulumi from "@pulumi/pulumi";
import * as k8s from "@pulumi/kubernetes";

export interface InfrastructureOptions {
    environment: string;
    enableLoki?: boolean;
    enableConsul?: boolean;
}

export interface PostgresOutput {
    serviceName: string | pulumi.Output<string>;
    secretName: string;
    port: number;
}

export interface RedisOutput {
    serviceName: string | pulumi.Output<string>;
    port: number;
}

export interface ConsulOutput {
    serviceName: pulumi.Output<string>;
    port: number;
}

export interface InfrastructureOutput {
    postgres: PostgresOutput;
    redis: RedisOutput;
    consul?: ConsulOutput;
    loki?: k8s.apps.v1.StatefulSet;
}

export function deployInfrastructure(
    namespace: string,
    options: InfrastructureOptions
): InfrastructureOutput {
    const labels = {
        "app.kubernetes.io/part-of": "aetherium",
        "environment": options.environment,
    };

    // PostgreSQL Secret
    const postgresPassword = pulumi.secret("aetherium-secret-password");
    const postgresSecret = new k8s.core.v1.Secret("postgres-secret", {
        metadata: {
            name: "postgres-credentials",
            namespace: namespace,
            labels,
        },
        type: "Opaque",
        stringData: postgresPassword.apply(pwd => ({
            "POSTGRES_USER": "aetherium",
            "POSTGRES_PASSWORD": pwd,
            "POSTGRES_DB": "aetherium",
        })),
    });

    // PostgreSQL StatefulSet
    const postgresStatefulSet = new k8s.apps.v1.StatefulSet("postgres", {
        metadata: {
            name: "postgres",
            namespace: namespace,
            labels: { ...labels, "app.kubernetes.io/name": "postgres" },
        },
        spec: {
            serviceName: "postgres",
            replicas: 1,
            selector: {
                matchLabels: { app: "postgres" },
            },
            template: {
                metadata: {
                    labels: { app: "postgres", ...labels },
                },
                spec: {
                    containers: [{
                        name: "postgres",
                        image: "postgres:15-alpine",
                        ports: [{ containerPort: 5432, name: "postgres" }],
                        envFrom: [{
                            secretRef: { name: "postgres-credentials" },
                        }],
                        volumeMounts: [{
                            name: "postgres-data",
                            mountPath: "/var/lib/postgresql/data",
                        }],
                        resources: {
                            requests: { memory: "256Mi", cpu: "250m" },
                            limits: { memory: "1Gi", cpu: "1000m" },
                        },
                        livenessProbe: {
                            exec: {
                                command: ["pg_isready", "-U", "aetherium"],
                            },
                            initialDelaySeconds: 30,
                            periodSeconds: 10,
                        },
                        readinessProbe: {
                            exec: {
                                command: ["pg_isready", "-U", "aetherium"],
                            },
                            initialDelaySeconds: 5,
                            periodSeconds: 5,
                        },
                    }],
                },
            },
            volumeClaimTemplates: [{
                metadata: { name: "postgres-data" },
                spec: {
                    accessModes: ["ReadWriteOnce"],
                    resources: {
                        requests: { storage: "10Gi" },
                    },
                },
            }],
        },
    });

    // PostgreSQL Service
    const postgresService = new k8s.core.v1.Service("postgres-service", {
        metadata: {
            name: "postgres",
            namespace: namespace,
            labels: { ...labels, "app.kubernetes.io/name": "postgres" },
        },
        spec: {
            type: "ClusterIP",
            ports: [{ port: 5432, targetPort: 5432, name: "postgres" }],
            selector: { app: "postgres" },
        },
    });

    // Redis StatefulSet
    const redisStatefulSet = new k8s.apps.v1.StatefulSet("redis", {
        metadata: {
            name: "redis",
            namespace: namespace,
            labels: { ...labels, "app.kubernetes.io/name": "redis" },
        },
        spec: {
            serviceName: "redis",
            replicas: 1,
            selector: {
                matchLabels: { app: "redis" },
            },
            template: {
                metadata: {
                    labels: { app: "redis", ...labels },
                },
                spec: {
                    containers: [{
                        name: "redis",
                        image: "redis:7-alpine",
                        ports: [{ containerPort: 6379, name: "redis" }],
                        command: ["redis-server", "--appendonly", "yes"],
                        volumeMounts: [{
                            name: "redis-data",
                            mountPath: "/data",
                        }],
                        resources: {
                            requests: { memory: "128Mi", cpu: "100m" },
                            limits: { memory: "512Mi", cpu: "500m" },
                        },
                        livenessProbe: {
                            exec: {
                                command: ["redis-cli", "ping"],
                            },
                            initialDelaySeconds: 30,
                            periodSeconds: 10,
                        },
                        readinessProbe: {
                            exec: {
                                command: ["redis-cli", "ping"],
                            },
                            initialDelaySeconds: 5,
                            periodSeconds: 5,
                        },
                    }],
                },
            },
            volumeClaimTemplates: [{
                metadata: { name: "redis-data" },
                spec: {
                    accessModes: ["ReadWriteOnce"],
                    resources: {
                        requests: { storage: "5Gi" },
                    },
                },
            }],
        },
    });

    // Redis Service
    const redisService = new k8s.core.v1.Service("redis-service", {
        metadata: {
            name: "redis",
            namespace: namespace,
            labels: { ...labels, "app.kubernetes.io/name": "redis" },
        },
        spec: {
            type: "ClusterIP",
            ports: [{ port: 6379, targetPort: 6379, name: "redis" }],
            selector: { app: "redis" },
        },
    });

    // ==========================================================================
    // Consul - Service Discovery for Workers
    // ==========================================================================
    let consul: ConsulOutput | undefined;
    if (options.enableConsul !== false) { // Enable by default
        const consulStatefulSet = new k8s.apps.v1.StatefulSet("consul", {
            metadata: {
                name: "consul",
                namespace: namespace,
                labels: { ...labels, "app.kubernetes.io/name": "consul" },
            },
            spec: {
                serviceName: "consul",
                replicas: 1,
                selector: {
                    matchLabels: { app: "consul" },
                },
                template: {
                    metadata: {
                        labels: { app: "consul", ...labels },
                    },
                    spec: {
                        containers: [{
                            name: "consul",
                            image: "hashicorp/consul:1.17",
                            ports: [
                                { containerPort: 8500, name: "http" },
                                { containerPort: 8600, name: "dns-tcp", protocol: "TCP" },
                                { containerPort: 8600, name: "dns-udp", protocol: "UDP" },
                                { containerPort: 8301, name: "serf-lan" },
                                { containerPort: 8302, name: "serf-wan" },
                                { containerPort: 8300, name: "server" },
                            ],
                            args: [
                                "agent",
                                "-server",
                                "-bootstrap-expect=1",
                                "-ui",
                                "-client=0.0.0.0",
                                "-data-dir=/consul/data",
                                "-datacenter=dc1",
                            ],
                            env: [{
                                name: "POD_IP",
                                valueFrom: {
                                    fieldRef: { fieldPath: "status.podIP" },
                                },
                            }],
                            volumeMounts: [{
                                name: "consul-data",
                                mountPath: "/consul/data",
                            }],
                            resources: {
                                requests: { memory: "128Mi", cpu: "100m" },
                                limits: { memory: "256Mi", cpu: "250m" },
                            },
                            livenessProbe: {
                                httpGet: {
                                    path: "/v1/status/leader",
                                    port: "http",
                                },
                                initialDelaySeconds: 30,
                                periodSeconds: 10,
                            },
                            readinessProbe: {
                                httpGet: {
                                    path: "/v1/status/leader",
                                    port: "http",
                                },
                                initialDelaySeconds: 10,
                                periodSeconds: 5,
                            },
                        }],
                    },
                },
                volumeClaimTemplates: [{
                    metadata: { name: "consul-data" },
                    spec: {
                        accessModes: ["ReadWriteOnce"],
                        resources: {
                            requests: { storage: "1Gi" },
                        },
                    },
                }],
            },
        });

        const consulService = new k8s.core.v1.Service("consul-service", {
            metadata: {
                name: "consul",
                namespace: namespace,
                labels: { ...labels, "app.kubernetes.io/name": "consul" },
            },
            spec: {
                type: "ClusterIP",
                ports: [
                    { port: 8500, targetPort: 8500, name: "http" },
                    { port: 8600, targetPort: 8600, name: "dns-tcp", protocol: "TCP" },
                    { port: 8600, targetPort: 8600, name: "dns-udp", protocol: "UDP" },
                ],
                selector: { app: "consul" },
            },
        });

        consul = {
            serviceName: consulService.metadata.name,
            port: 8500,
        };
    }

    // Optional: Loki for logging
    let loki: k8s.apps.v1.StatefulSet | undefined;
    if (options.enableLoki) {
        loki = new k8s.apps.v1.StatefulSet("loki", {
            metadata: {
                name: "loki",
                namespace: namespace,
                labels: { ...labels, "app.kubernetes.io/name": "loki" },
            },
            spec: {
                serviceName: "loki",
                replicas: 1,
                selector: {
                    matchLabels: { app: "loki" },
                },
                template: {
                    metadata: {
                        labels: { app: "loki", ...labels },
                    },
                    spec: {
                        containers: [{
                            name: "loki",
                            image: "grafana/loki:2.9.0",
                            ports: [{ containerPort: 3100, name: "http" }],
                            args: ["-config.file=/etc/loki/local-config.yaml"],
                            volumeMounts: [{
                                name: "loki-data",
                                mountPath: "/loki",
                            }],
                            resources: {
                                requests: { memory: "256Mi", cpu: "100m" },
                                limits: { memory: "1Gi", cpu: "500m" },
                            },
                        }],
                    },
                },
                volumeClaimTemplates: [{
                    metadata: { name: "loki-data" },
                    spec: {
                        accessModes: ["ReadWriteOnce"],
                        resources: {
                            requests: { storage: "10Gi" },
                        },
                    },
                }],
            },
        });

        new k8s.core.v1.Service("loki-service", {
            metadata: {
                name: "loki",
                namespace: namespace,
                labels: { ...labels, "app.kubernetes.io/name": "loki" },
            },
            spec: {
                type: "ClusterIP",
                ports: [{ port: 3100, targetPort: 3100, name: "http" }],
                selector: { app: "loki" },
            },
        });
    }

    // Return infrastructure outputs
    const postgresServiceName = postgresService.metadata.name;
    const redisServiceName = redisService.metadata.name;

    return {
        postgres: {
            serviceName: postgresServiceName,
            secretName: "postgres-credentials",
            port: 5432,
        },
        redis: {
            serviceName: redisServiceName,
            port: 6379,
        },
        consul,
        loki,
    };
}
