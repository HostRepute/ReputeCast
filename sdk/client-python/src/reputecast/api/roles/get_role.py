from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.role_detail import RoleDetail
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    role_id: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/roles/{role_id}".format(workspace=quote(str(workspace), safe=""),role_id=quote(str(role_id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | RoleDetail:
    if response.status_code == 200:
        response_200 = RoleDetail.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | RoleDetail]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    role_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | RoleDetail]:
    """ One role for the role editor

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        role_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | RoleDetail]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
role_id=role_id,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    role_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Problem | RoleDetail | None:
    """ One role for the role editor

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        role_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | RoleDetail
     """


    return sync_detailed(
        workspace=workspace,
role_id=role_id,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    role_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | RoleDetail]:
    """ One role for the role editor

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        role_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | RoleDetail]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
role_id=role_id,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    role_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Problem | RoleDetail | None:
    """ One role for the role editor

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        role_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | RoleDetail
     """


    return (await asyncio_detailed(
        workspace=workspace,
role_id=role_id,
client=client,

    )).parsed
