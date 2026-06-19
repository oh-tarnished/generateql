"""Asynchronous HTTP client with retry and timeout support."""

import asyncio
import time
from typing import override

import aiohttp
from loom.network._helpers.helpers import (
    build_full_url,
    pretty_print_json,
    validate_status_code,
)
from loom.network._shared.metrics import NetworkErrorMetrics, NetworkHTTPMetrics
from loom.network._shared.pulse import pulse
from loom.network.options.connection import ConnectionOptions, HTTPResponse, URLOptions
from pulse import TracedOperation, trace


class HTTPClient:
    """Async HTTP client providing retry, timeout, and sync wrapper support.

    This client mirrors Go-style HTTP client behavior using aiohttp.
    """

    def __init__(self, opts: ConnectionOptions | None = None):
        """Initialize the HTTP client.

        Sets up internal state for the aiohttp session and
        connection options. The client must be connected
        before making requests.
        """
        self.session: aiohttp.ClientSession | None = None
        self.opts: ConnectionOptions | None = opts

    @trace("http_connect", auto_events=True)
    async def connect(self, opts: ConnectionOptions | None = None) -> None:
        """Initialize the HTTP client session.

        Args:
            opts: Optional connection configuration options.

        Raises:
            ValueError: If URL scheme is not http or https or if no options are provided.
        """
        if opts:
            self.opts = opts

        if not self.opts:
            pulse.logger.error("HTTPClient connected without options")
            error_msg = "Connection options must be provided"
            raise ValueError(error_msg)
        try:
            if self.session:
                pulse.logger.debug("HTTP client already connected")
                return  # already connected

            if self.opts.url.scheme not in ("http", "https"):
                _raise_url_scheme_error(self.opts.url.scheme)

            pulse.logger.info(f"Connecting to HTTP endpoint at {self.opts.url.host}")

            timeout = aiohttp.ClientTimeout(total=self.opts.timeout)
            self.session = aiohttp.ClientSession(
                headers=self.opts.headers,
                timeout=timeout,
            )
            pulse.logger.debug(f"HTTP client session created with timeout: {self.opts.timeout}s")
        except Exception as e:
            pulse.metrics.record(
                NetworkErrorMetrics(connection_error_total=1),
                labels={"host": self.opts.url.host if self.opts else "unknown"},
            )
            pulse.logger.warning("Failed to initialize HTTP client: %s", e)
            error_msg = "Failed to initialize HTTP client"
            raise RuntimeError(error_msg) from e

    @trace("http_close", auto_events=True)
    async def close(self) -> None:
        """Close the HTTP client session."""
        if self.session:
            await self.session.close()
            pulse.logger.debug("HTTP client session closed")
        self.session = None

    @trace("http_reconnect", auto_events=True)
    async def reconnect(self) -> None:
        """Reinitialize the HTTP client using existing options.

        Raises:
            RuntimeError: If the client was not previously initialized.
        """
        try:
            if not self.opts:
                _raise_client_not_initialized_error()
            pulse.logger.info("Attempting to reconnect HTTP client")
            await self.close()
            await self.connect()
        except Exception as e:
            pulse.logger.warning("Failed to reconnect HTTP client: %s", e)
            error_msg = "Failed to reconnect HTTP client"
            raise RuntimeError(error_msg) from e

    async def _send_request(
        self,
        method: str,
        full_url: str,
        body: bytes | None,
        headers: dict[str, str] | None,
    ) -> bytes:
        """Send a single HTTP request and validate the response.

        Args:
            method: HTTP method.
            full_url: Fully qualified request URL.
            body: Optional request body.
            headers: Optional per-request headers.

        Returns:
            Raw response body bytes.

        Raises:
            RuntimeError: If the client is not connected or response is invalid.
        """
        start_time = time.perf_counter()
        labels = {
            "method": method,
            "host": str(full_url).split("//")[-1].split("/")[0],
        }

        try:
            if not self.session:
                await self.connect()

            pulse.logger.debug(f"Sending {method} request to {full_url}")
            with TracedOperation(
                pulse.tracing,
                "http_request",
                {"http.method": method, "http.url": full_url},
            ):
                async with self.session.request(
                    method=method,
                    url=full_url,
                    data=body,
                    headers=headers,
                ) as resp:
                    data = await resp.read()
                    duration = time.perf_counter() - start_time
                    pulse.logger.debug(f"Received response status: {resp.status}")

                    if "application/json" in resp.headers.get("Content-Type", ""):
                        try:
                            _ = pretty_print_json(data)
                        except ValueError:
                            pulse.logger.warning("Response contains invalid JSON")

                    pulse.metrics.record(
                        NetworkHTTPMetrics(
                            http_request_total=1,
                            http_request_duration_seconds=duration,
                            http_response_size_bytes=len(data),
                        ),
                        labels=labels,
                    )

                    validate_status_code(resp.status)
                    return data
        except asyncio.CancelledError:
            raise
        except Exception as e:
            pulse.metrics.record(NetworkErrorMetrics(http_error_total=1), labels=labels)
            pulse.logger.warning("HTTP request failed: %s", str(e))
            error_msg = "HTTP request failed"
            raise RuntimeError(error_msg) from e

    @override
    @trace("http_request_with_retry", auto_events=True)
    async def request(
        self,
        method: str,
        url_opts: URLOptions,
        body: bytes | None,
        headers: dict[str, str],
        path_index: int,
        max_retries: int,
    ) -> HTTPResponse:
        """Perform an HTTP request with retry support.

        Args:
            method: HTTP method.
            url_opts: URL configuration options.
            body: Optional request body.
            headers: Per-request headers.
            path_index: Index into URL paths.
            max_retries: Maximum retry attempts.

        Returns:
            HTTPResponse containing data or error.
        """
        if not self.session:
            await self.connect()
        retry_delay = self.opts.retry_delay if self.opts and self.opts.retry_delay else 2.0
        last_error: Exception | None = None

        pulse.logger.info(f"Starting request with max {max_retries} retries")
        for attempt in range(max_retries + 1):
            if attempt > 0:
                pulse.logger.info(f"Retry attempt {attempt} after {retry_delay}s delay")
                pulse.metrics.record(
                    NetworkHTTPMetrics(http_retry_total=1),
                    labels={"host": self.opts.url.host},
                )
                await asyncio.sleep(retry_delay)

            try:
                full_url = build_full_url(url_opts, path_index)
                return await self._send_request(
                    method=method,
                    full_url=full_url,
                    body=body,
                    headers=headers or None,
                )

            except asyncio.CancelledError:
                raise

            except (
                ConnectionError,
                TimeoutError,
                ValueError,
                TypeError,
                aiohttp.ClientError,
            ) as e:
                last_error = e
                pulse.logger.debug(f"Request attempt {attempt + 1} failed: {e}")

        pulse.logger.warning("Request failed after %d retries", max_retries)
        error_msg = f"Request failed after {max_retries} retries"

        class MaxRetriesExceededError(Exception):
            pass

        raise MaxRetriesExceededError(error_msg) from last_error

    @override
    @trace("http_request_sync", auto_events=True)
    def request_sync(
        self,
        method: str,
        url_opts: URLOptions,
        body: bytes | None,
        headers: dict[str, str],
        path_index: int,
        max_retries: int,
    ) -> bytes:
        """Perform a synchronous HTTP request.

        Args:
            method: HTTP method.
            url_opts: URL configuration options.
            body: Optional request body.
            headers: Per-request headers.
            path_index: Index into URL paths.
            max_retries: Maximum retry attempts.

        Returns:
            Raw response body bytes.

        Raises:
            Exception: If the request fails.
        """

        async def _run():
            pulse.logger.debug("Executing synchronous HTTP request")
            try:
                resp = await self.request(
                    method,
                    url_opts,
                    body,
                    headers,
                    path_index,
                    max_retries,
                )
            except Exception as e:
                pulse.logger.warning("Synchronous request failed: %s", e)
                raise
            return resp

        return asyncio.run(_run())


def _raise_url_scheme_error(scheme: str) -> None:
    """Raise URL scheme error."""
    pulse.logger.warning("Invalid URL scheme: %s", scheme)
    error_msg = "URL scheme must be http or https"
    raise ValueError(error_msg)


def _raise_client_not_initialized_error() -> None:
    """Raise client not initialized error."""
    pulse.logger.warning("Client not initialized for reconnect")
    error_msg = "Client not initialized"
    raise RuntimeError(error_msg)
