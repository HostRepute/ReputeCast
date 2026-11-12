from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.get_integration_app_kind import GetIntegrationAppKind
from ...models.integration_app import IntegrationApp
from ...models.problem import Problem
from typing import cast



def _get_kwargs(
    workspace: str,
    kind: GetIntegrationAppKind,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/integrations/apps/{kind}".format(workspace=quote(str(workspace), safe=""),kind=quote(str(kind), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> IntegrationApp | Problem:
    if response.status_code == 200:
        response_200 = IntegrationApp.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[IntegrationApp | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    kind: GetIntegrationAppKind,
    *,
    client: AuthenticatedClient,

) -> Response[IntegrationApp | Problem]:
    """ An automation platform's page: numbers, workflows, webhooks, what it can do and its key

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        kind (GetIntegrationAppKind):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[IntegrationApp | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
kind=kind,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    kind: GetIntegrationAppKind,
    *,
    client: AuthenticatedClient,

) -> IntegrationApp | Problem | None:
    """ An automation platform's page: numbers, workflows, webhooks, what it can do and its key

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        kind (GetIntegrationAppKind):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        IntegrationApp | Problem
     """


    return sync_detailed(
        workspace=workspace,
kind=kind,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    kind: GetIntegrationAppKind,
    *,
    client: AuthenticatedClient,

) -> Response[IntegrationApp | Problem]:
    """ An automation platform's page: numbers, workflows, webhooks, what it can do and its key

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        kind (GetIntegrationAppKind):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[IntegrationApp | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
kind=kind,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    kind: GetIntegrationAppKind,
    *,
    client: AuthenticatedClient,

) -> IntegrationApp | Problem | None:
    """ An automation platform's page: numbers, workflows, webhooks, what it can do and its key

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        kind (GetIntegrationAppKind):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        IntegrationApp | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
kind=kind,
client=client,

    )).parsed
