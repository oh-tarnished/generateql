"""Shared network configuration models.

This module defines:
- Client and protocol enums
- URL construction options
- Generic connection configuration
- Protocol-specific result containers

The models here are intentionally lightweight and reusable across
HTTP, GraphQL, and future WebSocket clients.
"""

from dataclasses import dataclass, field
from enum import StrEnum
from typing import Any


class ClientType(StrEnum):
    """Supported network client types."""

    HTTP = "http"
    GRAPHQL = "graphql"
    WEBSOCKET = "websocket"


class URLScheme(StrEnum):
    """Supported URL schemes."""

    HTTP = "http"
    HTTPS = "https"
    WS = "ws"
    WSS = "wss"


class HTTPMethod(StrEnum):
    """Supported HTTP request methods."""

    GET = "GET"
    POST = "POST"
    PUT = "PUT"
    PATCH = "PATCH"
    DELETE = "DELETE"


@dataclass
class URLOptions:
    """URL configuration used to construct request endpoints.

    Attributes:
    scheme : URLScheme
        URL scheme (http, https, ws, wss).
    host : str
        Hostname with optional port (e.g. "localhost:8080").
    paths : list[str]
        Optional list of endpoint paths.
    params : dict[str, str]
        Optional query parameters.
    """

    scheme: URLScheme
    host: str
    paths: list[str] = field(default_factory=list)
    params: dict[str, str] = field(default_factory=dict)

    def base(self) -> str:
        """Return the base URL composed of scheme and host.

        Returns:
        str
            Base URL (e.g. "https://example.com").
        """
        return f"{self.scheme.value}://{self.host}"


@dataclass
class ConnectionOptions:
    """Generic connection configuration shared by network clients.

    Attributes:
    url : URLOptions
        URL configuration for the connection.
    timeout : float
        Request timeout in seconds.
    headers : dict[str, str]
        Default headers applied to all requests.
    retries : int
        Number of retry attempts on failure.
    retry_delay : float
        Delay between retry attempts in seconds.
    """

    url: URLOptions
    timeout: float = 30.0
    headers: dict[str, str] = field(default_factory=dict)
    retries: int = 0
    retry_delay: float = 2.0


@dataclass
class HTTPResponse:
    """Container for HTTP request results.

    Attributes:
    data : bytes | None
        Raw response payload.
    error : Exception | None
        Error raised during request execution, if any.
    """

    data: bytes | None = None
    error: Exception | None = None


@dataclass
class GQLQueryResult:
    """Result of a GraphQL query execution.

    Attributes:
    response : Any | None
        Parsed GraphQL response data.
    error : Exception | None
        Error raised during execution, if any.
    """

    response: Any | None = None
    error: Exception | None = None


@dataclass
class GQLMutationResult:
    """Result of a GraphQL mutation execution.

    Attributes:
    response : Any | None
        Parsed GraphQL response data.
    error : Exception | None
        Error raised during execution, if any.
    """

    response: Any | None = None
    error: Exception | None = None
