from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.provider_held_decision_input import ProviderHeldDecisionInput
from ...models.provider_held_decision_result import ProviderHeldDecisionResult
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    provider_id: UUID,
    campaign_id: UUID,
    *,
    body: ProviderHeldDecisionInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/providers/{provider_id}/held/{campaign_id}".format(workspace=quote(str(workspace), safe=""),provider_id=quote(str(provider_id), safe=""),campaign_id=quote(str(campaign_id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | ProviderHeldDecisionResult:
    if response.status_code == 200:
        response_200 = ProviderHeldDecisionResult.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | ProviderHeldDecisionResult]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    provider_id: UUID,
    campaign_id: UUID,
    *,
    client: AuthenticatedClient,
    body: ProviderHeldDecisionInput,

) -> Response[Problem | ProviderHeldDecisionResult]:
    """ Wait, move or cancel a broadcast share held on a provider that is down

     Cancelling needs campaigns:pause; waiting for the provider and moving the held recipients resume the
    send and need campaigns:send.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        provider_id (UUID):
        campaign_id (UUID):
        body (ProviderHeldDecisionInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ProviderHeldDecisionResult]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
provider_id=provider_id,
campaign_id=campaign_id,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    provider_id: UUID,
    campaign_id: UUID,
    *,
    client: AuthenticatedClient,
    body: ProviderHeldDecisionInput,

) -> Problem | ProviderHeldDecisionResult | None:
    """ Wait, move or cancel a broadcast share held on a provider that is down

     Cancelling needs campaigns:pause; waiting for the provider and moving the held recipients resume the
    send and need campaigns:send.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        provider_id (UUID):
        campaign_id (UUID):
        body (ProviderHeldDecisionInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ProviderHeldDecisionResult
     """


    return sync_detailed(
        workspace=workspace,
provider_id=provider_id,
campaign_id=campaign_id,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    provider_id: UUID,
    campaign_id: UUID,
    *,
    client: AuthenticatedClient,
    body: ProviderHeldDecisionInput,

) -> Response[Problem | ProviderHeldDecisionResult]:
    """ Wait, move or cancel a broadcast share held on a provider that is down

     Cancelling needs campaigns:pause; waiting for the provider and moving the held recipients resume the
    send and need campaigns:send.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        provider_id (UUID):
        campaign_id (UUID):
        body (ProviderHeldDecisionInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ProviderHeldDecisionResult]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
provider_id=provider_id,
campaign_id=campaign_id,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    provider_id: UUID,
    campaign_id: UUID,
    *,
    client: AuthenticatedClient,
    body: ProviderHeldDecisionInput,

) -> Problem | ProviderHeldDecisionResult | None:
    """ Wait, move or cancel a broadcast share held on a provider that is down

     Cancelling needs campaigns:pause; waiting for the provider and moving the held recipients resume the
    send and need campaigns:send.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        provider_id (UUID):
        campaign_id (UUID):
        body (ProviderHeldDecisionInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ProviderHeldDecisionResult
     """


    return (await asyncio_detailed(
        workspace=workspace,
provider_id=provider_id,
campaign_id=campaign_id,
client=client,
body=body,

    )).parsed
