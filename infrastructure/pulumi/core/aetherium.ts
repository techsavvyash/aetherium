import * as pulumi from "@pulumi/pulumi";
import * as k8s from "@pulumi/kubernetes";
import { PostgresOutput, RedisOutput } from "./infrastructure";

export interface AetheriumOptions {
    environment: string;
    postgres: PostgresOutput;
    redis: RedisOutput;
    imageTag?: string;
    workerReplicas?: number;
}

export interface AetheriumOutput {
    apiGateway: k8s.apps.v1.Deployment;
    worker: k8s.apps.v1.DaemonSet;
    helmRelease: k8s.helm.v3.Release;
}

export function deployAetherium(
    namespace: pulumi.Output<string>,
    options: AetheriumOptions
): AetheriumOutput {
    const imageTag = options.imageTag || "latest";
    const isProd = options.environment === "production";

    // Deploy Aetherium using Helm chart
    const helmRelease = new k8s.helm.v3.Release("aetherium", {
        name: "aetherium",
        namespace: namespace,
        chart: "../helm/aetherium",
        values: {
            global: {
                environment: options.environment,
                imageRegistry: "ghcr.io/techsavvyash",
            },
            apiGateway: {
                enabled: true,
                replicaCount: isProd ? 3 : 1,
                image: {
                    repository: "aetherium/api-gateway",
                    tag: imageTag,
                },
                service: {
                    type: isProd ? "LoadBalancer" : "ClusterIP",
                    port: 8080,
                },
                resources: {
                    requests: {
                        memory: isProd ? "256Mi" : "128Mi",
                        cpu: isProd ? "250m" : "100m",
                    },
                    limits: {
                        memory: isProd ? "512Mi" : "256Mi",
                        cpu: isProd ? "500m" : "250m",
                    },
                },
            },
            worker: {
                enabled: true,
                // DaemonSet for workers - one per node with KVM support
                kind: "DaemonSet",
                image: {
                    repository: "aetherium/worker",
                    tag: imageTag,
                },
                // Workers need privileged access for Firecracker
                securityContext: {
                    privileged: true,
                },
                nodeSelector: {
                    "aetherium.io/kvm-enabled": "true",
                },
                tolerations: [{
                    key: "aetherium.io/worker",
                    operator: "Exists",
                    effect: "NoSchedule",
                }],
                hostNetwork: true,
                resources: {
                    requests: {
                        memory: "512Mi",
                        cpu: "500m",
                    },
                    limits: {
                        memory: "2Gi",
                        cpu: "2000m",
                    },
                },
                volumeMounts: [{
                    name: "dev-kvm",
                    mountPath: "/dev/kvm",
                }, {
                    name: "firecracker-data",
                    mountPath: "/var/firecracker",
                }],
                volumes: [{
                    name: "dev-kvm",
                    hostPath: {
                        path: "/dev/kvm",
                        type: "CharDevice",
                    },
                }, {
                    name: "firecracker-data",
                    hostPath: {
                        path: "/var/firecracker",
                        type: "DirectoryOrCreate",
                    },
                }],
            },
            postgresql: {
                enabled: false, // Using our own PostgreSQL from infrastructure.ts
                external: {
                    host: "postgres",
                    port: options.postgres.port,
                    existingSecret: options.postgres.secretName,
                },
            },
            redis: {
                enabled: false, // Using our own Redis from infrastructure.ts
                external: {
                    host: "redis",
                    port: options.redis.port,
                },
            },
            config: {
                vmm: {
                    provider: "firecracker",
                    firecracker: {
                        kernelPath: "/var/firecracker/vmlinux",
                        rootfsTemplate: "/var/firecracker/rootfs.ext4",
                    },
                },
                queue: {
                    provider: "asynq",
                    concurrency: isProd ? 20 : 5,
                },
                logging: {
                    provider: isProd ? "loki" : "stdout",
                    lokiUrl: isProd ? "http://loki:3100" : "",
                },
            },
        },
    });

    // Get references to deployed resources
    const apiGateway = k8s.apps.v1.Deployment.get(
        "api-gateway-deployment",
        pulumi.interpolate`${namespace}/aetherium-api-gateway`
    );

    const worker = k8s.apps.v1.DaemonSet.get(
        "worker-daemonset",
        pulumi.interpolate`${namespace}/aetherium-worker`
    );

    return {
        apiGateway,
        worker,
        helmRelease,
    };
}
