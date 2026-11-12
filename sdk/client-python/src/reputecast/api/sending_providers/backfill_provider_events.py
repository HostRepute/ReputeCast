from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.provider_backfill_result import ProviderBackfillResult
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    provider_id: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/providers/{provider_id}/backfill".format(workspace=quote(str(workspace), safe=""),provider_id=quote(str(provider_id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | ProviderBackfillResult:
    if response.status_code == 202:
        response_202 = ProviderBackfillResult.from_dict(response.json())



        return response_202

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | ProviderBackfillResult]:
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

) -> Response[Problem | ProviderBackfillResult]:
    """ Pull message status from the provider's API

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        provider_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ProviderBackfillResult]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
provider_id=provider_id,

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

) -> Problem | ProviderBackfillResult | None:
    """ Pull message status from the provider's API

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        provider_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ProviderBackfillResult
     """


    return sync_detailed(
        workspace=workspace,
provider_id=provider_id,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    provider_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | ProviderBackfillResult]:
    """ Pull message status from the provider's API

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        provider_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ProviderBackfillResult]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
provider_id=provider_id,

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

) -> Problem | ProviderBackfillResult | None:
    """ Pull message status from the provider's API

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        provider_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ProviderBackfillResult
     """


    return (await asyncio_detailed(
        workspace=workspace,
provider_id=provider_id,
client=client,

    )).parsed
