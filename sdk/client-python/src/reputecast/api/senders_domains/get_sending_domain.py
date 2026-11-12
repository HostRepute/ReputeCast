from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.sending_domain_detail import SendingDomainDetail
from ...types import UNSET, Unset
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    domain_id: UUID,
    *,
    provider: str | Unset = UNSET,
    refresh: bool | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["provider"] = provider

    params["refresh"] = refresh


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/domains/{domain_id}".format(workspace=quote(str(workspace), safe=""),domain_id=quote(str(domain_id), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | SendingDomainDetail:
    if response.status_code == 200:
        response_200 = SendingDomainDetail.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | SendingDomainDetail]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    domain_id: UUID,
    *,
    client: AuthenticatedClient,
    provider: str | Unset = UNSET,
    refresh: bool | Unset = UNSET,

) -> Response[Problem | SendingDomainDetail]:
    """ A sending domain's DNS records with their live status

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        domain_id (UUID):
        provider (str | Unset): Only the records this provider needs (and the domain-wide ones)
        refresh (bool | Unset): Check DNS again first when the last check is older than 20 seconds

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SendingDomainDetail]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
domain_id=domain_id,
provider=provider,
refresh=refresh,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    domain_id: UUID,
    *,
    client: AuthenticatedClient,
    provider: str | Unset = UNSET,
    refresh: bool | Unset = UNSET,

) -> Problem | SendingDomainDetail | None:
    """ A sending domain's DNS records with their live status

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        domain_id (UUID):
        provider (str | Unset): Only the records this provider needs (and the domain-wide ones)
        refresh (bool | Unset): Check DNS again first when the last check is older than 20 seconds

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SendingDomainDetail
     """


    return sync_detailed(
        workspace=workspace,
domain_id=domain_id,
client=client,
provider=provider,
refresh=refresh,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    domain_id: UUID,
    *,
    client: AuthenticatedClient,
    provider: str | Unset = UNSET,
    refresh: bool | Unset = UNSET,

) -> Response[Problem | SendingDomainDetail]:
    """ A sending domain's DNS records with their live status

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        domain_id (UUID):
        provider (str | Unset): Only the records this provider needs (and the domain-wide ones)
        refresh (bool | Unset): Check DNS again first when the last check is older than 20 seconds

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SendingDomainDetail]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
domain_id=domain_id,
provider=provider,
refresh=refresh,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    domain_id: UUID,
    *,
    client: AuthenticatedClient,
    provider: str | Unset = UNSET,
    refresh: bool | Unset = UNSET,

) -> Problem | SendingDomainDetail | None:
    """ A sending domain's DNS records with their live status

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        domain_id (UUID):
        provider (str | Unset): Only the records this provider needs (and the domain-wide ones)
        refresh (bool | Unset): Check DNS again first when the last check is older than 20 seconds

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SendingDomainDetail
     """


    return (await asyncio_detailed(
        workspace=workspace,
domain_id=domain_id,
client=client,
provider=provider,
refresh=refresh,

    )).parsed
