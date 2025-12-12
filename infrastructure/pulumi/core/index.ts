import * as pulumi from "@pulumi/pulumi";
import * as k8s from "@pulumi/kubernetes";
import { createNamespace } from "./namespace";
import { deployInfrastructure } from "./infrastructure";
import { deployAetherium } from "./aetherium";

// Configuration
const config = new pulumi.Config();
const environment = config.get("environment") || "development";
const clusterConfig = new pulumi.Config("kubernetes");
const cloudConfig = new pulumi.Config("cloud");

const clusterName = clusterConfig.get("clusterName") || "aetherium-cluster";
const provider = cloudConfig.get("provider") || "local";

// Create namespace
const namespace = createNamespace("aetherium", {
    environment,
    labels: {
        "app.kubernetes.io/managed-by": "pulumi",
        "app.kubernetes.io/part-of": "aetherium",
    },
});

// Get namespace name - extract from Output<string>
const namespaceName = namespace.metadata.name.apply(n => n);

// Deploy infrastructure components (Redis, PostgreSQL, Loki)
const infra = pulumi.all([namespaceName]).apply(([nsName]) =>
    deployInfrastructure(nsName, {
        environment,
        enableLoki: environment === "production",
        enableConsul: environment === "production",
    })
);

// Deploy Aetherium services using Helm
const aetherium = pulumi.all([namespaceName, infra]).apply(([nsName, infraOutput]) =>
    deployAetherium(nsName, {
        environment,
        postgres: infraOutput.postgres,
        redis: infraOutput.redis,
        imageTag: environment === "production" ? "latest" : "dev",
    })
);

// Export outputs
export const outputs = pulumi.all([namespaceName, infra, aetherium]).apply(([nsName, infraOutput, aetheriumOutput]) => ({
    namespace: nsName,
    environment,
    clusterName,
    provider,
    infrastructure: {
        postgres: {
            serviceName: infraOutput.postgres.serviceName,
            port: infraOutput.postgres.port,
        },
        redis: {
            serviceName: infraOutput.redis.serviceName,
            port: infraOutput.redis.port,
        },
    },
    aetherium: {
        helmReleaseId: aetheriumOutput.helmRelease.id,
        helmReleaseName: aetheriumOutput.helmRelease.name,
    },
}));
