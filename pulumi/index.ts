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

// Main deployment
async function main() {
    // Create namespace
    const namespace = createNamespace("aetherium", {
        environment,
        labels: {
            "app.kubernetes.io/managed-by": "pulumi",
            "app.kubernetes.io/part-of": "aetherium",
        },
    });

    // Deploy infrastructure components (Redis, PostgreSQL, Loki)
    const infra = deployInfrastructure(namespace.metadata.name, {
        environment,
        enableLoki: environment === "production",
        enableConsul: environment === "production",
    });

    // Deploy Aetherium services using Helm
    const aetherium = deployAetherium(namespace.metadata.name, {
        environment,
        postgres: infra.postgres,
        redis: infra.redis,
        imageTag: environment === "production" ? "latest" : "dev",
    });

    return {
        namespace: namespace.metadata.name,
        environment,
        clusterName,
        provider,
        services: {
            apiGateway: aetherium.apiGateway.status,
            worker: aetherium.worker.status,
        },
        endpoints: {
            postgres: infra.postgres.serviceName,
            redis: infra.redis.serviceName,
        },
    };
}

// Export outputs
export const outputs = main();
