from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.contact_activity_filter import ContactActivityFilter
from ...models.contact_activity_page import ContactActivityPage
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    contact: UUID,
    *,
    filter_: ContactActivityFilter | Unset = ContactActivityFilter.ALL,
    cursor: str | Unset = UNSET,
    limit: int | Unset = 8,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    json_filter_: str | Unset = UNSET
    if not isinstance(filter_, Unset):
        json_filter_ = filter_.value

    params["filter"] = json_filter_

    params["cursor"] = cursor

    params["limit"] = limit


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/contacts/{contact}/activity".format(workspace=quote(str(workspace), safe=""),contact=quote(str(contact), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> ContactActivityPage | Problem:
    if response.status_code == 200:
        response_200 = ContactActivityPage.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[ContactActivityPage | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    contact: UUID,
    *,
    client: AuthenticatedClient,
    filter_: ContactActivityFilter | Unset = ContactActivityFilter.ALL,
    cursor: str | Unset = UNSET,
    limit: int | Unset = 8,

) -> Response[ContactActivityPage | Problem]:
    """ Activity timeline, including which provider delivered each email

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact (UUID):
        filter_ (ContactActivityFilter | Unset):  Default: ContactActivityFilter.ALL.
        cursor (str | Unset):
        limit (int | Unset):  Default: 8.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ContactActivityPage | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
contact=contact,
filter_=filter_,
cursor=cursor,
limit=limit,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    contact: UUID,
    *,
    client: AuthenticatedClient,
    filter_: ContactActivityFilter | Unset = ContactActivityFilter.ALL,
    cursor: str | Unset = UNSET,
    limit: int | Unset = 8,

) -> ContactActivityPage | Problem | None:
    """ Activity timeline, including which provider delivered each email

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact (UUID):
        filter_ (ContactActivityFilter | Unset):  Default: ContactActivityFilter.ALL.
        cursor (str | Unset):
        limit (int | Unset):  Default: 8.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ContactActivityPage | Problem
     """


    return sync_detailed(
        workspace=workspace,
contact=contact,
client=client,
filter_=filter_,
cursor=cursor,
limit=limit,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    contact: UUID,
    *,
    client: AuthenticatedClient,
    filter_: ContactActivityFilter | Unset = ContactActivityFilter.ALL,
    cursor: str | Unset = UNSET,
    limit: int | Unset = 8,

) -> Response[ContactActivityPage | Problem]:
    """ Activity timeline, including which provider delivered each email

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact (UUID):
        filter_ (ContactActivityFilter | Unset):  Default: ContactActivityFilter.ALL.
        cursor (str | Unset):
        limit (int | Unset):  Default: 8.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ContactActivityPage | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
contact=contact,
filter_=filter_,
cursor=cursor,
limit=limit,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    contact: UUID,
    *,
    client: AuthenticatedClient,
    filter_: ContactActivityFilter | Unset = ContactActivityFilter.ALL,
    cursor: str | Unset = UNSET,
    limit: int | Unset = 8,

) -> ContactActivityPage | Problem | None:
    """ Activity timeline, including which provider delivered each email

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        contact (UUID):
        filter_ (ContactActivityFilter | Unset):  Default: ContactActivityFilter.ALL.
        cursor (str | Unset):
        limit (int | Unset):  Default: 8.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ContactActivityPage | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
contact=contact,
client=client,
filter_=filter_,
cursor=cursor,
limit=limit,

    )).parsed
