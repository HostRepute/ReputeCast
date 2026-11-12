from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.connect_test_status import ConnectTestStatus
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    setup_id: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/connect/setups/{setup_id}/test".format(workspace=quote(str(workspace), safe=""),setup_id=quote(str(setup_id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> ConnectTestStatus | Problem:
    if response.status_code == 200:
        response_200 = ConnectTestStatus.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[ConnectTestStatus | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    setup_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[ConnectTestStatus | Problem]:
    """ The test send form and the delivery timeline of the latest test

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        setup_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ConnectTestStatus | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
setup_id=setup_id,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    setup_id: UUID,
    *,
    client: AuthenticatedClient,

) -> ConnectTestStatus | Problem | None:
    """ The test send form and the delivery timeline of the latest test

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        setup_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ConnectTestStatus | Problem
     """


    return sync_detailed(
        workspace=workspace,
setup_id=setup_id,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    setup_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[ConnectTestStatus | Problem]:
    """ The test send form and the delivery timeline of the latest test

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        setup_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ConnectTestStatus | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
setup_id=setup_id,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    setup_id: UUID,
    *,
    client: AuthenticatedClient,

) -> ConnectTestStatus | Problem | None:
    """ The test send form and the delivery timeline of the latest test

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        setup_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ConnectTestStatus | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
setup_id=setup_id,
client=client,

    )).parsed
