from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.import_detail import ImportDetail
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    id: UUID,
    *,
    slow: bool | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["slow"] = slow


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/imports/{id}/resume".format(workspace=quote(str(workspace), safe=""),id=quote(str(id), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> ImportDetail | Problem:
    if response.status_code == 200:
        response_200 = ImportDetail.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[ImportDetail | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    id: UUID,
    *,
    client: AuthenticatedClient,
    slow: bool | Unset = UNSET,

) -> Response[ImportDetail | Problem]:
    """ Resume an import paused by the source's rate limit

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        slow (bool | Unset): Pace requests to the source to 3 a second from now on

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ImportDetail | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
slow=slow,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    id: UUID,
    *,
    client: AuthenticatedClient,
    slow: bool | Unset = UNSET,

) -> ImportDetail | Problem | None:
    """ Resume an import paused by the source's rate limit

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        slow (bool | Unset): Pace requests to the source to 3 a second from now on

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ImportDetail | Problem
     """


    return sync_detailed(
        workspace=workspace,
id=id,
client=client,
slow=slow,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    id: UUID,
    *,
    client: AuthenticatedClient,
    slow: bool | Unset = UNSET,

) -> Response[ImportDetail | Problem]:
    """ Resume an import paused by the source's rate limit

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        slow (bool | Unset): Pace requests to the source to 3 a second from now on

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ImportDetail | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
slow=slow,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    id: UUID,
    *,
    client: AuthenticatedClient,
    slow: bool | Unset = UNSET,

) -> ImportDetail | Problem | None:
    """ Resume an import paused by the source's rate limit

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        slow (bool | Unset): Pace requests to the source to 3 a second from now on

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ImportDetail | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
id=id,
client=client,
slow=slow,

    )).parsed
