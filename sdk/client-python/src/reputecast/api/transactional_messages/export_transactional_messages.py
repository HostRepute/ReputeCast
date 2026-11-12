from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.export_transactional_messages_range import ExportTransactionalMessagesRange
from ...models.export_transactional_messages_status import ExportTransactionalMessagesStatus
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    status: ExportTransactionalMessagesStatus | Unset = ExportTransactionalMessagesStatus.ALL,
    template: str | Unset = UNSET,
    provider: str | Unset = UNSET,
    key: str | Unset = UNSET,
    range_: ExportTransactionalMessagesRange | Unset = ExportTransactionalMessagesRange.VALUE_1,
    from_: str | Unset = UNSET,
    to: str | Unset = UNSET,
    q: str | Unset = UNSET,
    before: str | Unset = UNSET,
    after: str | Unset = UNSET,
    limit: int | Unset = 25,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    json_status: str | Unset = UNSET
    if not isinstance(status, Unset):
        json_status = status.value

    params["status"] = json_status

    params["template"] = template

    params["provider"] = provider

    params["key"] = key

    json_range_: str | Unset = UNSET
    if not isinstance(range_, Unset):
        json_range_ = range_.value

    params["range"] = json_range_

    params["from"] = from_

    params["to"] = to

    params["q"] = q

    params["before"] = before

    params["after"] = after

    params["limit"] = limit


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/transactional/messages/export".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | str:
    if response.status_code == 200:
        response_200 = cast(str, response.json())
        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | str]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    status: ExportTransactionalMessagesStatus | Unset = ExportTransactionalMessagesStatus.ALL,
    template: str | Unset = UNSET,
    provider: str | Unset = UNSET,
    key: str | Unset = UNSET,
    range_: ExportTransactionalMessagesRange | Unset = ExportTransactionalMessagesRange.VALUE_1,
    from_: str | Unset = UNSET,
    to: str | Unset = UNSET,
    q: str | Unset = UNSET,
    before: str | Unset = UNSET,
    after: str | Unset = UNSET,
    limit: int | Unset = 25,

) -> Response[Problem | str]:
    """ Export the filtered log as CSV

     Holds recipient addresses, so it needs transactional:content. Up to 50,000 rows; every export is
    audited.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ExportTransactionalMessagesStatus | Unset):  Default:
            ExportTransactionalMessagesStatus.ALL.
        template (str | Unset): A template's ID, or raw for mail that arrived by SMTP
        provider (str | Unset): A sending provider's ID
        key (str | Unset): An API key's ID
        range_ (ExportTransactionalMessagesRange | Unset):  Default:
            ExportTransactionalMessagesRange.VALUE_1.
        from_ (str | Unset): First day of a custom range (YYYY-MM-DD, the workspace's time zone),
            within the last 90 days
        to (str | Unset): Last day of a custom range
        q (str | Unset): Recipient, message ID, idempotency key, tag or metadata value
        before (str | Unset): next of the previous page: older messages
        after (str | Unset): newest of a page: only newer messages (live updates)
        limit (int | Unset):  Default: 25.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | str]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
status=status,
template=template,
provider=provider,
key=key,
range_=range_,
from_=from_,
to=to,
q=q,
before=before,
after=after,
limit=limit,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    status: ExportTransactionalMessagesStatus | Unset = ExportTransactionalMessagesStatus.ALL,
    template: str | Unset = UNSET,
    provider: str | Unset = UNSET,
    key: str | Unset = UNSET,
    range_: ExportTransactionalMessagesRange | Unset = ExportTransactionalMessagesRange.VALUE_1,
    from_: str | Unset = UNSET,
    to: str | Unset = UNSET,
    q: str | Unset = UNSET,
    before: str | Unset = UNSET,
    after: str | Unset = UNSET,
    limit: int | Unset = 25,

) -> Problem | str | None:
    """ Export the filtered log as CSV

     Holds recipient addresses, so it needs transactional:content. Up to 50,000 rows; every export is
    audited.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ExportTransactionalMessagesStatus | Unset):  Default:
            ExportTransactionalMessagesStatus.ALL.
        template (str | Unset): A template's ID, or raw for mail that arrived by SMTP
        provider (str | Unset): A sending provider's ID
        key (str | Unset): An API key's ID
        range_ (ExportTransactionalMessagesRange | Unset):  Default:
            ExportTransactionalMessagesRange.VALUE_1.
        from_ (str | Unset): First day of a custom range (YYYY-MM-DD, the workspace's time zone),
            within the last 90 days
        to (str | Unset): Last day of a custom range
        q (str | Unset): Recipient, message ID, idempotency key, tag or metadata value
        before (str | Unset): next of the previous page: older messages
        after (str | Unset): newest of a page: only newer messages (live updates)
        limit (int | Unset):  Default: 25.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | str
     """


    return sync_detailed(
        workspace=workspace,
client=client,
status=status,
template=template,
provider=provider,
key=key,
range_=range_,
from_=from_,
to=to,
q=q,
before=before,
after=after,
limit=limit,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    status: ExportTransactionalMessagesStatus | Unset = ExportTransactionalMessagesStatus.ALL,
    template: str | Unset = UNSET,
    provider: str | Unset = UNSET,
    key: str | Unset = UNSET,
    range_: ExportTransactionalMessagesRange | Unset = ExportTransactionalMessagesRange.VALUE_1,
    from_: str | Unset = UNSET,
    to: str | Unset = UNSET,
    q: str | Unset = UNSET,
    before: str | Unset = UNSET,
    after: str | Unset = UNSET,
    limit: int | Unset = 25,

) -> Response[Problem | str]:
    """ Export the filtered log as CSV

     Holds recipient addresses, so it needs transactional:content. Up to 50,000 rows; every export is
    audited.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ExportTransactionalMessagesStatus | Unset):  Default:
            ExportTransactionalMessagesStatus.ALL.
        template (str | Unset): A template's ID, or raw for mail that arrived by SMTP
        provider (str | Unset): A sending provider's ID
        key (str | Unset): An API key's ID
        range_ (ExportTransactionalMessagesRange | Unset):  Default:
            ExportTransactionalMessagesRange.VALUE_1.
        from_ (str | Unset): First day of a custom range (YYYY-MM-DD, the workspace's time zone),
            within the last 90 days
        to (str | Unset): Last day of a custom range
        q (str | Unset): Recipient, message ID, idempotency key, tag or metadata value
        before (str | Unset): next of the previous page: older messages
        after (str | Unset): newest of a page: only newer messages (live updates)
        limit (int | Unset):  Default: 25.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | str]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
status=status,
template=template,
provider=provider,
key=key,
range_=range_,
from_=from_,
to=to,
q=q,
before=before,
after=after,
limit=limit,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    status: ExportTransactionalMessagesStatus | Unset = ExportTransactionalMessagesStatus.ALL,
    template: str | Unset = UNSET,
    provider: str | Unset = UNSET,
    key: str | Unset = UNSET,
    range_: ExportTransactionalMessagesRange | Unset = ExportTransactionalMessagesRange.VALUE_1,
    from_: str | Unset = UNSET,
    to: str | Unset = UNSET,
    q: str | Unset = UNSET,
    before: str | Unset = UNSET,
    after: str | Unset = UNSET,
    limit: int | Unset = 25,

) -> Problem | str | None:
    """ Export the filtered log as CSV

     Holds recipient addresses, so it needs transactional:content. Up to 50,000 rows; every export is
    audited.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ExportTransactionalMessagesStatus | Unset):  Default:
            ExportTransactionalMessagesStatus.ALL.
        template (str | Unset): A template's ID, or raw for mail that arrived by SMTP
        provider (str | Unset): A sending provider's ID
        key (str | Unset): An API key's ID
        range_ (ExportTransactionalMessagesRange | Unset):  Default:
            ExportTransactionalMessagesRange.VALUE_1.
        from_ (str | Unset): First day of a custom range (YYYY-MM-DD, the workspace's time zone),
            within the last 90 days
        to (str | Unset): Last day of a custom range
        q (str | Unset): Recipient, message ID, idempotency key, tag or metadata value
        before (str | Unset): next of the previous page: older messages
        after (str | Unset): newest of a page: only newer messages (live updates)
        limit (int | Unset):  Default: 25.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | str
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
status=status,
template=template,
provider=provider,
key=key,
range_=range_,
from_=from_,
to=to,
q=q,
before=before,
after=after,
limit=limit,

    )).parsed
