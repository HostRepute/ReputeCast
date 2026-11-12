from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.integration_dead_letter_list import IntegrationDeadLetterList
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    id: UUID,
    webhook_id: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/integrations/connections/{id}/webhooks/{webhook_id}/dead-letters".format(workspace=quote(str(workspace), safe=""),id=quote(str(id), safe=""),webhook_id=quote(str(webhook_id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> IntegrationDeadLetterList | Problem:
    if response.status_code == 200:
        response_200 = IntegrationDeadLetterList.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[IntegrationDeadLetterList | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    id: UUID,
    webhook_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[IntegrationDeadLetterList | Problem]:
    """ A managed webhook's dead-letter queue

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        webhook_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[IntegrationDeadLetterList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
webhook_id=webhook_id,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    id: UUID,
    webhook_id: UUID,
    *,
    client: AuthenticatedClient,

) -> IntegrationDeadLetterList | Problem | None:
    """ A managed webhook's dead-letter queue

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        webhook_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        IntegrationDeadLetterList | Problem
     """


    return sync_detailed(
        workspace=workspace,
id=id,
webhook_id=webhook_id,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    id: UUID,
    webhook_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[IntegrationDeadLetterList | Problem]:
    """ A managed webhook's dead-letter queue

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        webhook_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[IntegrationDeadLetterList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
webhook_id=webhook_id,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    id: UUID,
    webhook_id: UUID,
    *,
    client: AuthenticatedClient,

) -> IntegrationDeadLetterList | Problem | None:
    """ A managed webhook's dead-letter queue

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        webhook_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        IntegrationDeadLetterList | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
id=id,
webhook_id=webhook_id,
client=client,

    )).parsed
