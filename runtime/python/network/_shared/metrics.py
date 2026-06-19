"""Metrics definitions for network operations."""

import pulse
from pulse import MetricsBaseModel

# Buckets for network latency (ms to 30s)
NETWORK_LATENCY_BUCKETS = [
    0.001,
    0.005,
    0.01,
    0.025,
    0.05,
    0.1,
    0.25,
    0.5,
    1.0,
    2.5,
    5.0,
    10.0,
    20.0,
    30.0,
]

# Buckets for response sizes (bytes)
NETWORK_SIZE_BUCKETS = [100, 1000, 10000, 100000, 1000000, 10000000]


class NetworkHTTPMetrics(MetricsBaseModel):
    """HTTP client metrics."""

    http_request_total: int = pulse.Counter(description="Total HTTP requests sent")
    http_request_duration_seconds: float = pulse.Histogram(
        description="HTTP request duration in seconds", buckets=NETWORK_LATENCY_BUCKETS
    )
    http_response_size_bytes: int = pulse.Histogram(
        description="HTTP response size in bytes", buckets=NETWORK_SIZE_BUCKETS
    )
    http_retry_total: int = pulse.Counter(description="Total number of HTTP retries")


class NetworkGraphQLMetrics(MetricsBaseModel):
    """GraphQL client metrics."""

    graphql_query_total: int = pulse.Counter(description="Total GraphQL queries executed")
    graphql_mutation_total: int = pulse.Counter(description="Total GraphQL mutations executed")
    graphql_duration_seconds: float = pulse.Histogram(
        description="GraphQL operation duration in seconds",
        buckets=NETWORK_LATENCY_BUCKETS,
    )


class NetworkErrorMetrics(MetricsBaseModel):
    """Network-related error metrics."""

    http_error_total: int = pulse.Counter(description="Total HTTP request errors")
    graphql_error_total: int = pulse.Counter(description="Total GraphQL operation errors")
    connection_error_total: int = pulse.Counter(description="Total connection establishment errors")
