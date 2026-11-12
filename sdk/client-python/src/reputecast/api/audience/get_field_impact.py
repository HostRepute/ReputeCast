from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.field_impact import FieldImpact
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    id: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/fields/{id}/impact".format(workspace=quote(str(workspace), safe=""),id=quote(str(id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> FieldImpact | Problem:
    if response.status_code == 200:
        response_200 = FieldImpact.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[FieldImpact | Problem]:
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

) -> Response[FieldImpact | Problem]:
    """ What deleting a custom field would affect

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[FieldImpact | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,

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

) -> FieldImpact | Problem | None:
    """ What deleting a custom field would affect

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        FieldImpact | Problem
     """


    return sync_detailed(
        workspace=workspace,
id=id,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[FieldImpact | Problem]:
    """ What deleting a custom field would affect

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[FieldImpact | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,

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

) -> FieldImpact | Problem | None:
    """ What deleting a custom field would affect

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        FieldImpact | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
id=id,
client=client,

    )).parsed
