from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.list_suppressions_applies_to import ListSuppressionsAppliesTo
from ...models.list_suppressions_reason import ListSuppressionsReason
from ...models.list_suppressions_since import ListSuppressionsSince
from ...models.problem import Problem
from ...models.suppression_list import SuppressionList
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    q: str | Unset = UNSET,
    reason: ListSuppressionsReason | Unset = UNSET,
    source: str | Unset = UNSET,
    since: ListSuppressionsSince | Unset = UNSET,
    import_: str | Unset = UNSET,
    applies_to: ListSuppressionsAppliesTo | Unset = UNSET,
    cursor: str | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 10,
    exact: bool | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["q"] = q

    json_reason: str | Unset = UNSET
    if not isinstance(reason, Unset):
        json_reason = reason.value

    params["reason"] = json_reason

    params["source"] = source

    json_since: str | Unset = UNSET
    if not isinstance(since, Unset):
        json_since = since.value

    params["since"] = json_since

    params["import"] = import_

    json_applies_to: str | Unset = UNSET
    if not isinstance(applies_to, Unset):
        json_applies_to = applies_to.value

    params["appliesTo"] = json_applies_to

    params["cursor"] = cursor

    params["page"] = page

    params["pageSize"] = page_size

    params["exact"] = exact


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/suppressions".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | SuppressionList:
    if response.status_code == 200:
        response_200 = SuppressionList.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | SuppressionList]:
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
    q: str | Unset = UNSET,
    reason: ListSuppressionsReason | Unset = UNSET,
    source: str | Unset = UNSET,
    since: ListSuppressionsSince | Unset = UNSET,
    import_: str | Unset = UNSET,
    applies_to: ListSuppressionsAppliesTo | Unset = UNSET,
    cursor: str | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 10,
    exact: bool | Unset = UNSET,

) -> Response[Problem | SuppressionList]:
    """ The shared suppression list with reasons, sources and provider sync

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): Address or domain
        reason (ListSuppressionsReason | Unset):
        source (str | Unset):
        since (ListSuppressionsSince | Unset):
        import_ (str | Unset): Show the rows of one import group
        applies_to (ListSuppressionsAppliesTo | Unset): all: every stream; marketing: Broadcasts
            and Automations only (unsubscribes, role addresses)
        cursor (str | Unset): nextCursor or prevCursor of the page before
        page (int | Unset): A page number from the start, when there is no cursor Default: 1.
        page_size (int | Unset):  Default: 10.
        exact (bool | Unset): Count past 10,000

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SuppressionList]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
q=q,
reason=reason,
source=source,
since=since,
import_=import_,
applies_to=applies_to,
cursor=cursor,
page=page,
page_size=page_size,
exact=exact,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    q: str | Unset = UNSET,
    reason: ListSuppressionsReason | Unset = UNSET,
    source: str | Unset = UNSET,
    since: ListSuppressionsSince | Unset = UNSET,
    import_: str | Unset = UNSET,
    applies_to: ListSuppressionsAppliesTo | Unset = UNSET,
    cursor: str | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 10,
    exact: bool | Unset = UNSET,

) -> Problem | SuppressionList | None:
    """ The shared suppression list with reasons, sources and provider sync

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): Address or domain
        reason (ListSuppressionsReason | Unset):
        source (str | Unset):
        since (ListSuppressionsSince | Unset):
        import_ (str | Unset): Show the rows of one import group
        applies_to (ListSuppressionsAppliesTo | Unset): all: every stream; marketing: Broadcasts
            and Automations only (unsubscribes, role addresses)
        cursor (str | Unset): nextCursor or prevCursor of the page before
        page (int | Unset): A page number from the start, when there is no cursor Default: 1.
        page_size (int | Unset):  Default: 10.
        exact (bool | Unset): Count past 10,000

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SuppressionList
     """


    return sync_detailed(
        workspace=workspace,
client=client,
q=q,
reason=reason,
source=source,
since=since,
import_=import_,
applies_to=applies_to,
cursor=cursor,
page=page,
page_size=page_size,
exact=exact,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    q: str | Unset = UNSET,
    reason: ListSuppressionsReason | Unset = UNSET,
    source: str | Unset = UNSET,
    since: ListSuppressionsSince | Unset = UNSET,
    import_: str | Unset = UNSET,
    applies_to: ListSuppressionsAppliesTo | Unset = UNSET,
    cursor: str | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 10,
    exact: bool | Unset = UNSET,

) -> Response[Problem | SuppressionList]:
    """ The shared suppression list with reasons, sources and provider sync

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): Address or domain
        reason (ListSuppressionsReason | Unset):
        source (str | Unset):
        since (ListSuppressionsSince | Unset):
        import_ (str | Unset): Show the rows of one import group
        applies_to (ListSuppressionsAppliesTo | Unset): all: every stream; marketing: Broadcasts
            and Automations only (unsubscribes, role addresses)
        cursor (str | Unset): nextCursor or prevCursor of the page before
        page (int | Unset): A page number from the start, when there is no cursor Default: 1.
        page_size (int | Unset):  Default: 10.
        exact (bool | Unset): Count past 10,000

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SuppressionList]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
q=q,
reason=reason,
source=source,
since=since,
import_=import_,
applies_to=applies_to,
cursor=cursor,
page=page,
page_size=page_size,
exact=exact,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    q: str | Unset = UNSET,
    reason: ListSuppressionsReason | Unset = UNSET,
    source: str | Unset = UNSET,
    since: ListSuppressionsSince | Unset = UNSET,
    import_: str | Unset = UNSET,
    applies_to: ListSuppressionsAppliesTo | Unset = UNSET,
    cursor: str | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 10,
    exact: bool | Unset = UNSET,

) -> Problem | SuppressionList | None:
    """ The shared suppression list with reasons, sources and provider sync

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): Address or domain
        reason (ListSuppressionsReason | Unset):
        source (str | Unset):
        since (ListSuppressionsSince | Unset):
        import_ (str | Unset): Show the rows of one import group
        applies_to (ListSuppressionsAppliesTo | Unset): all: every stream; marketing: Broadcasts
            and Automations only (unsubscribes, role addresses)
        cursor (str | Unset): nextCursor or prevCursor of the page before
        page (int | Unset): A page number from the start, when there is no cursor Default: 1.
        page_size (int | Unset):  Default: 10.
        exact (bool | Unset): Count past 10,000

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SuppressionList
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
q=q,
reason=reason,
source=source,
since=since,
import_=import_,
applies_to=applies_to,
cursor=cursor,
page=page,
page_size=page_size,
exact=exact,

    )).parsed
