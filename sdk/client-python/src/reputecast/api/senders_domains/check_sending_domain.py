from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.sending_domain_detail import SendingDomainDetail
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    domain_id: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/domains/{domain_id}/check".format(workspace=quote(str(workspace), safe=""),domain_id=quote(str(domain_id), safe=""),),
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

) -> Response[Problem | SendingDomainDetail]:
    """ Check a sending domain's DNS now

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        domain_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SendingDomainDetail]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
domain_id=domain_id,

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

) -> Problem | SendingDomainDetail | None:
    """ Check a sending domain's DNS now

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        domain_id (UUID):

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

    ).parsed

async def asyncio_detailed(
    workspace: str,
    domain_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | SendingDomainDetail]:
    """ Check a sending domain's DNS now

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        domain_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SendingDomainDetail]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
domain_id=domain_id,

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

) -> Problem | SendingDomainDetail | None:
    """ Check a sending domain's DNS now

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        domain_id (UUID):

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

    )).parsed
