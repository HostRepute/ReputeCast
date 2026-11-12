"""The ReputeCast API client for Python.

    import os
    from reputecast import ReputeCast, unwrap
    from reputecast.api.audience import list_lists

    rc = ReputeCast(api_key=os.environ["REPUTECAST_KEY"], base_url="https://news.example.com")
    lists = unwrap(list_lists.sync_detailed(client=rc, workspace="acme-newsletter"))

Every operation an API key may call is a module of reputecast.api, by tag,
with sync, sync_detailed, asyncio and asyncio_detailed functions; their
bodies and answers are the classes of reputecast.models. Docs:
https://github.com/hostrepute/reputecast/blob/main/docs/api/README.md#client-sdks
"""

from ._client import MAX_RETRY_DELAY, RETRY_STATUSES, USER_AGENT, ReputeCast, RetryTransport, is_retryable, retry_delay
from ._errors import ReputeCastError, unwrap
from ._version import __version__
from .client import AuthenticatedClient, Client
from .types import UNSET, File, Response, Unset

__all__ = [
    "MAX_RETRY_DELAY",
    "RETRY_STATUSES",
    "UNSET",
    "USER_AGENT",
    "AuthenticatedClient",
    "Client",
    "File",
    "ReputeCast",
    "ReputeCastError",
    "Response",
    "RetryTransport",
    "Unset",
    "__version__",
    "is_retryable",
    "retry_delay",
    "unwrap",
]
