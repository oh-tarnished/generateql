"""main network implementation module."""

from __future__ import annotations

from loom.network._shared.pulse import pulse
from loom.network.graphql import GraphQLClient
from loom.network.http import HTTPClient
from loom.network.options import ClientType, ConnectionOptions
from pulse import trace

NetworkClient = HTTPClient | GraphQLClient


class Network:
    """High-level network connection manager.

    Responsible for:
    - Creating the appropriate client (HTTP or GraphQL)
    - Managing connection lifecycle
    - Exposing the concrete client in a safe, typed manner
    """

    def __init__(self, client: NetworkClient) -> None:
        """Initialize the network clinet."""
        self._client: NetworkClient = client
        self._options: ConnectionOptions | None = None

    @classmethod
    @trace("network_create", auto_events=True)
    def create(cls, client_type: ClientType) -> Network:
        """Create a Network instance for the given client type.

        Parameters
        ----------
        client_type : ClientType
            Type of client to create.

        Returns:
        -------
        Network
            Configured Network instance (not yet connected).
        """
        pulse.logger.info("Creating network client with type: %s", client_type)
        if client_type == ClientType.HTTP:
            client = HTTPClient()
            pulse.logger.debug("HTTP client created")
        elif client_type == ClientType.GRAPHQL:
            client = GraphQLClient()
            pulse.logger.debug("GraphQL client created")
        else:
            pulse.logger.warning("Unsupported client type: %s", client_type)
            error_msg = f"Unsupported client type: {client_type}"
            raise ValueError(error_msg)

        return cls(client)

    @trace("network_connect", auto_events=True)
    async def connect(self, options: ConnectionOptions) -> None:
        """Configure and connect the underlying client.

        Parameters
        ----------
        options : ConnectionOptions
            Connection configuration.
        """
        pulse.logger.info(f"Connecting network client to {options.url.host}")
        self._options = options
        await self._client.connect(options)
        pulse.logger.debug("Network client connected successfully")

    @trace("network_reconnect", auto_events=True)
    async def reconnect(self) -> None:
        """Reconnect using previously provided connection options.

        Raises:
        ------
        RuntimeError
            If the client has not been connected before.
        """
        if not self._options:
            pulse.logger.warning("Cannot reconnect before initial connect")
            error_msg = "Cannot reconnect before initial connect"
            raise RuntimeError(error_msg)
        pulse.logger.info("Reconnecting network client")
        await self._client.reconnect()

    @trace("network_close", auto_events=True)
    async def close(self) -> None:
        """Close the underlying client connection."""
        pulse.logger.info("Closing network client connection")
        await self._client.close()
        pulse.logger.debug("Network client closed")

    @property
    def client(self) -> NetworkClient:
        """Return the underlying concrete client.

        Returns:
        -------
        HTTPClient | GraphQLClient
            The active client instance.
        """
        return self._client

    def http(self) -> HTTPClient:
        """Return the client as an HTTPClient.

        Raises:
        ------
        TypeError
            If the underlying client is not HTTP.
        """
        if not isinstance(self._client, HTTPClient):
            error_msg = "Network client is not an HTTPClient"
            raise TypeError(error_msg)
        return self._client

    def graphql(self) -> GraphQLClient:
        """Return the client as a GraphQLClient.

        Raises:
        ------
        TypeError
            If the underlying client is not GraphQL.
        """
        if not isinstance(self._client, GraphQLClient):
            error_msg = "Network client is not a GraphQLClient"
            raise TypeError(error_msg)
        return self._client
