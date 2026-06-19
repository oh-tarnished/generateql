"""Asynchronous GraphQL client built on gql + aiohttp."""

from __future__ import annotations

import time
from typing import TYPE_CHECKING, Any, Self

from gql import Client, gql
from gql.transport.aiohttp import AIOHTTPTransport
from pulse import TracedOperation, trace

if TYPE_CHECKING:
    from graphql import DocumentNode

from loom.network._helpers.helpers import _build_args
from loom.network._shared.metrics import NetworkErrorMetrics, NetworkGraphQLMetrics
from loom.network._shared.pulse import pulse
from loom.network.options.connection import (
    ConnectionOptions,
    GQLMutationResult,
    GQLQueryResult,
)


class GraphQLClient:
    """Asynchronous GraphQL client with explicit lifecycle management.

    This client mirrors the behavior of the Go GraphQL client while
    following Python async/await conventions.
    """

    def __init__(self) -> None:
        """Initialize the GraphQL client.

        The client must be connected before executing queries or mutations.
        """
        self._client: Client | None = None
        self._session = None
        self._options: ConnectionOptions | None = None

    @trace("graphql_connect", auto_events=True)
    async def connect(self, options: ConnectionOptions) -> None:
        """Establish a connection to the GraphQL server.

        Args:
            options: GraphQL connection configuration.

        Raises:
            RuntimeError: If the connection fails.
        """
        self._options = options
        pulse.logger.info(f"Connecting to GraphQL server at {options.url.host}")

        # Build full URL from URLOptions
        url_opts = options.url

        if not url_opts.paths:
            pulse.logger.warning("GraphQL requires at least one path")
            error_msg = "GraphQL requires at least one path"
            raise ValueError(error_msg)

        full_url = f"{url_opts.scheme.value}://{url_opts.host}{url_opts.paths[0]}"

        transport = AIOHTTPTransport(
            url=full_url,
            timeout=options.timeout,
            headers=options.headers or {},
        )

        self._client = Client(
            transport=transport,
            fetch_schema_from_transport=False,
        )

        try:
            self._session = await self._client.connect_async()
            pulse.logger.debug(f"GraphQL client successfully connected to {full_url}")
        except Exception as e:
            pulse.metrics.record(
                NetworkErrorMetrics(connection_error_total=1),
                labels={"host": options.url.host},
            )
            pulse.logger.warning("Failed to connect to GraphQL server: %s", e)
            msg = f"Failed to connect to GraphQL server: {e}"
            raise RuntimeError(msg) from e

    @trace("graphql_reconnect", auto_events=True)
    async def reconnect(self) -> None:
        """Reconnect using the previously provided connection options.

        Raises:
            RuntimeError: If the client has not been initialized.
        """
        if not self._options:
            pulse.logger.warning("GraphQL client not initialized")
            error_msg = "GraphQL client not initialized"
            raise RuntimeError(error_msg)

        pulse.logger.info("Attempting to reconnect GraphQL client")
        await self.close()
        await self.connect(self._options)

    @trace("graphql_close", auto_events=True)
    async def close(self) -> None:
        """Close the GraphQL connection and release resources."""
        if self._client:
            await self._client.close_async()
            pulse.logger.debug("GraphQL client connection closed")

        self._client = None
        self._session = None

    @trace("graphql_query", auto_events=True)
    async def query(
        self,
        query: DocumentNode,
        variables: dict[str, Any] | None = None,
    ) -> GQLQueryResult:
        """Execute a GraphQL query.

        Args:
            query: Parsed GraphQL query document.
            variables: Optional query variables.

        Returns:
            GQLQueryResult containing response data or error.
        """
        if not self._session:
            err = RuntimeError("GraphQL client not connected")
            return GQLQueryResult(error=err)

        start_time = time.perf_counter()
        labels = {"host": self._options.url.host if self._options else "unknown"}

        try:
            pulse.logger.debug(f"Executing GraphQL query with variables: {variables}")
            with TracedOperation(pulse.tracing, "graphql_query_execute"):
                response = await self._session.execute(
                    query,
                    variable_values=variables,
                )
            duration = time.perf_counter() - start_time
            pulse.metrics.record(
                NetworkGraphQLMetrics(graphql_query_total=1, graphql_duration_seconds=duration),
                labels=labels,
            )
            pulse.logger.debug("GraphQL query executed successfully")
            return GQLQueryResult(response=response)

        except (ConnectionError, TimeoutError, ValueError, TypeError) as e:
            pulse.metrics.record(NetworkErrorMetrics(graphql_error_total=1), labels=labels)
            pulse.logger.warning("GraphQL query failed: %s", e)
            return GQLQueryResult(error=e)

    @trace("graphql_mutation", auto_events=True)
    async def mutation(
        self,
        mutation: DocumentNode,
        variables: dict[str, Any] | None = None,
    ) -> GQLMutationResult:
        """Execute a GraphQL mutation.

        Args:
            mutation: Parsed GraphQL mutation document.
            variables: Optional mutation variables.

        Returns:
            GQLMutationResult containing response data or error.
        """
        if not self._session:
            err = RuntimeError("GraphQL client not connected")
            return GQLMutationResult(error=err)

        start_time = time.perf_counter()
        labels = {"host": self._options.url.host if self._options else "unknown"}

        try:
            pulse.logger.debug(f"Executing GraphQL mutation with variables: {variables}")
            with TracedOperation(pulse.tracing, "graphql_mutation_execute"):
                response = await self._session.execute(
                    mutation,
                    variable_values=variables,
                )
            duration = time.perf_counter() - start_time
            pulse.metrics.record(
                NetworkGraphQLMetrics(graphql_mutation_total=1, graphql_duration_seconds=duration),
                labels=labels,
            )
            pulse.logger.debug("GraphQL mutation executed successfully")
            return GQLMutationResult(response=response)

        except (ConnectionError, TimeoutError, ValueError, TypeError) as e:
            pulse.metrics.record(NetworkErrorMetrics(graphql_error_total=1), labels=labels)
            pulse.logger.warning("GraphQL mutation failed: %s", e)
            return GQLMutationResult(error=e)

    @trace("graphql_mutation_with_input", auto_events=True)
    async def mutation_with_input(
        self,
        mutation_name: str,
        input_data: dict[str, Any],
        selection_set: str,
    ) -> GQLMutationResult:
        """Execute a mutation with inline input arguments.

        This mirrors the Go MutationWithInput helper.

        Args:
            mutation_name: Name of the mutation.
            input_data: Input arguments as a dictionary.
            selection_set: GraphQL selection set.

        Returns:
            GQLMutationResult containing response data or error.
        """
        try:
            pulse.logger.debug(f"Building mutation with name: {mutation_name}")
            args = _build_args(input_data)

            mutation_str = f"""
            mutation {{
                {mutation_name}({args}) {{
                    {selection_set}
                }}
            }}
            """

            mutation_doc = gql(mutation_str)

            return await self.mutation(mutation_doc)

        except (ValueError, TypeError) as e:
            pulse.logger.warning("Mutation with input failed: %s", e)
            return GQLMutationResult(error=e)

    @trace("graphql_query_str", auto_events=True)
    async def query_str(
        self,
        query: str,
        variables: dict[str, Any] | None = None,
    ):
        """Execute a GraphQL query from a raw string."""
        doc: DocumentNode = gql(query)
        return await self.query(doc, variables)

    @trace("graphql_mutation_str", auto_events=True)
    async def mutation_str(
        self,
        mutation: str,
        variables: dict[str, Any] | None = None,
    ):
        """Execute a GraphQL mutation from a raw string."""
        doc: DocumentNode = gql(mutation)
        return await self.mutation(doc, variables)

    async def __aenter__(self) -> Self:
        """Enter async context."""
        return self

    async def __aexit__(self, exc_type, exc, tb) -> None:
        """Exit async context and close connection."""
        await self.close()
