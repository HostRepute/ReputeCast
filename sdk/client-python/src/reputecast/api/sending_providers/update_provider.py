from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.provider_card import ProviderCard
from ...models.provider_settings_input import ProviderSettingsInput
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    provider_id: UUID,
    *,
    body: ProviderSettingsInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "patch",
        "url": "/api/v1/workspaces/{workspace}/providers/{provider_id}".format(workspace=quote(str(workspace), safe=""),provider_id=quote(str(provider_id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | ProviderCard:
    if response.status_code == 200:
        response_200 = ProviderCard.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | ProviderCard]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    provider_id: UUID,
    *,
    client: AuthenticatedClient,
    body: ProviderSettingsInput,

) -> Response[Problem | ProviderCard]:
    """ Change a provider's label, pricing or limits

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        provider_id (UUID):
        body (ProviderSettingsInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ProviderCard]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
provider_id=provider_id,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    provider_id: UUID,
    *,
    client: AuthenticatedClient,
    body: ProviderSettingsInput,

) -> Problem | ProviderCard | None:
    """ Change a provider's label, pricing or limits

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        provider_id (UUID):
        body (ProviderSettingsInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ProviderCard
     """


    return sync_detailed(
        workspace=workspace,
provider_id=provider_id,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    provider_id: UUID,
    *,
    client: AuthenticatedClient,
    body: ProviderSettingsInput,

) -> Response[Problem | ProviderCard]:
    """ Change a provider's label, pricing or limits

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        provider_id (UUID):
        body (ProviderSettingsInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ProviderCard]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
provider_id=provider_id,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    provider_id: UUID,
    *,
    client: AuthenticatedClient,
    body: ProviderSettingsInput,

) -> Problem | ProviderCard | None:
    """ Change a provider's label, pricing or limits

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        provider_id (UUID):
        body (ProviderSettingsInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ProviderCard
     """


    return (await asyncio_detailed(
        workspace=workspace,
provider_id=provider_id,
client=client,
body=body,

    )).parsed
