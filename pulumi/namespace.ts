import * as k8s from "@pulumi/kubernetes";

export interface NamespaceOptions {
    environment: string;
    labels?: Record<string, string>;
}

export function createNamespace(
    name: string,
    options: NamespaceOptions
): k8s.core.v1.Namespace {
    return new k8s.core.v1.Namespace(name, {
        metadata: {
            name: name,
            labels: {
                "app.kubernetes.io/name": name,
                "environment": options.environment,
                ...options.labels,
            },
            annotations: {
                "description": "Aetherium distributed task execution platform",
            },
        },
    });
}
