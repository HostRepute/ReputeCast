from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.integration_status import IntegrationStatus
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    id: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/integrations/connections/{id}/status".format(workspace=quote(str(workspace), safe=""),id=quote(str(id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> IntegrationStatus | Problem:
    if response.status_code == 200:
        response_200 = IntegrationStatus.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[IntegrationStatus | Problem]:
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

) -> Response[IntegrationStatus | Problem]:
    """ Whether a new key has been called yet (the connect dialog's last step)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[IntegrationStatus | Problem]
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

) -> IntegrationStatus | Problem | None:
    """ Whether a new key has been called yet (the connect dialog's last step)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        IntegrationStatus | Problem
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

) -> Response[IntegrationStatus | Problem]:
    """ Whether a new key has been called yet (the connect dialog's last step)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[IntegrationStatus | Problem]
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

) -> IntegrationStatus | Problem | None:
    """ Whether a new key has been called yet (the connect dialog's last step)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        IntegrationStatus | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
id=id,
client=client,

    )).parsed
