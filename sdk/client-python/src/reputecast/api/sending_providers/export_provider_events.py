from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.export_provider_events_type import ExportProviderEventsType
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    provider: str | Unset = UNSET,
    type_: ExportProviderEventsType | Unset = UNSET,
    q: str | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 25,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["provider"] = provider

    json_type_: str | Unset = UNSET
    if not isinstance(type_, Unset):
        json_type_ = type_.value

    params["type"] = json_type_

    params["q"] = q

    params["page"] = page

    params["pageSize"] = page_size


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/provider-events/export".format(workspace=quote(str(workspace), safe=""),),
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
    provider: str | Unset = UNSET,
    type_: ExportProviderEventsType | Unset = UNSET,
    q: str | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 25,

) -> Response[Problem | str]:
    """ Export the provider event log as CSV

     Full recipient addresses need subscribers:export; otherwise they are masked. Every export is
    audited.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        provider (str | Unset): Sending provider id
        type_ (ExportProviderEventsType | Unset):
        q (str | Unset): Recipient or provider message id
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 25.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | str]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
provider=provider,
type_=type_,
q=q,
page=page,
page_size=page_size,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    provider: str | Unset = UNSET,
    type_: ExportProviderEventsType | Unset = UNSET,
    q: str | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 25,

) -> Problem | str | None:
    """ Export the provider event log as CSV

     Full recipient addresses need subscribers:export; otherwise they are masked. Every export is
    audited.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        provider (str | Unset): Sending provider id
        type_ (ExportProviderEventsType | Unset):
        q (str | Unset): Recipient or provider message id
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 25.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | str
     """


    return sync_detailed(
        workspace=workspace,
client=client,
provider=provider,
type_=type_,
q=q,
page=page,
page_size=page_size,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    provider: str | Unset = UNSET,
    type_: ExportProviderEventsType | Unset = UNSET,
    q: str | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 25,

) -> Response[Problem | str]:
    """ Export the provider event log as CSV

     Full recipient addresses need subscribers:export; otherwise they are masked. Every export is
    audited.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        provider (str | Unset): Sending provider id
        type_ (ExportProviderEventsType | Unset):
        q (str | Unset): Recipient or provider message id
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 25.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | str]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
provider=provider,
type_=type_,
q=q,
page=page,
page_size=page_size,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    provider: str | Unset = UNSET,
    type_: ExportProviderEventsType | Unset = UNSET,
    q: str | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 25,

) -> Problem | str | None:
    """ Export the provider event log as CSV

     Full recipient addresses need subscribers:export; otherwise they are masked. Every export is
    audited.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        provider (str | Unset): Sending provider id
        type_ (ExportProviderEventsType | Unset):
        q (str | Unset): Recipient or provider message id
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 25.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | str
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
provider=provider,
type_=type_,
q=q,
page=page,
page_size=page_size,

    )).parsed
