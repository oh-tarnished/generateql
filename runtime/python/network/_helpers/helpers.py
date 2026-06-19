"""Helper utilities for HTTP network operations."""

import json
from typing import Any
from urllib.parse import urljoin

from loom.network._helpers.http_status import (
    HTTP_BAD_REQUEST,
    HTTP_FORBIDDEN,
    HTTP_NOT_FOUND,
    HTTP_OK_MAX,
    HTTP_OK_MIN,
    HTTP_SERVER_ERROR_MAX,
    HTTP_SERVER_ERROR_MIN,
    HTTP_TOO_MANY_REQUESTS,
    HTTP_UNAUTHORIZED,
)
from loom.network.options.connection import URLOptions


def build_full_url(url_opts: URLOptions, path_index: int) -> str:
    """Build a full request URL from URL options and path index.

    Args:
        url_opts: URL configuration containing scheme, host, and paths.
        path_index: Index of the path to append from url_opts.paths.

    Returns:
        Fully qualified URL string.

    Raises:
        ValueError: If path_index is out of range.
    """
    base = url_opts.base()

    if url_opts.paths:
        if path_index >= len(url_opts.paths):
            error_msg = "pathIndex out of range"
            raise ValueError(error_msg)
        return urljoin(base + "/", url_opts.paths[path_index].lstrip("/"))

    return base


def validate_status_code(status: int) -> None:
    """Validate HTTP response status code.

    Args:
        status: HTTP status code.

    Raises:
        RuntimeError: If the status code indicates an error.
    """
    if HTTP_OK_MIN <= status < HTTP_OK_MAX:
        return
    if status == HTTP_BAD_REQUEST:
        error_msg = "400 Bad Request"
        raise RuntimeError(error_msg)
    if status == HTTP_UNAUTHORIZED:
        error_msg = "401 Unauthorized"
        raise RuntimeError(error_msg)
    if status == HTTP_FORBIDDEN:
        error_msg = "403 Forbidden"
        raise RuntimeError(error_msg)
    if status == HTTP_NOT_FOUND:
        error_msg = "404 Not Found"
        raise RuntimeError(error_msg)
    if status == HTTP_TOO_MANY_REQUESTS:
        error_msg = "429 Too Many Requests"
        raise RuntimeError(error_msg)
    if HTTP_SERVER_ERROR_MIN <= status < HTTP_SERVER_ERROR_MAX:
        error_msg = f"{status} Server Error"
        raise RuntimeError(error_msg)
    error_msg = f"Unexpected status code: {status}"
    raise RuntimeError(error_msg)


def pretty_print_json(data: bytes) -> str:
    """Pretty-format a JSON byte payload.

    Raises:
        ValueError: If the data is not valid JSON.
    """
    try:
        obj = json.loads(data)
        return json.dumps(obj, indent=2)
    except json.JSONDecodeError as e:
        error_msg = "error formatting JSON"
        raise ValueError(error_msg) from e


def _build_args(variables: dict[str, Any]) -> str:
    """Build a GraphQL argument string from a dictionary.

    Args:
        variables: Input arguments.

    Returns:
        GraphQL-formatted argument string.
    """
    args: list[str] = []

    for key, value in variables.items():
        if isinstance(value, str):
            args.append(f'{key}: "{value}"')
        elif isinstance(value, bool):
            args.append(f"{key}: {'true' if value else 'false'}")
        elif isinstance(value, int | float):
            args.append(f"{key}: {value}")
        elif value is None:
            args.append(f"{key}: null")
        else:
            args.append(f"{key}: {json.dumps(value)}")

    return ", ".join(args)
