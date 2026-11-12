from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.partner_request_log import PartnerRequestLog
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    link: UUID,
    *,
    page: int | Unset = 1,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["page"] = page


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/grow/partner-links/{link}/requests".format(workspace=quote(str(workspace), safe=""),link=quote(str(link), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> PartnerRequestLog | Problem:
    if response.status_code == 200:
        response_200 = PartnerRequestLog.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[PartnerRequestLog | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    link: UUID,
    *,
    client: AuthenticatedClient,
    page: int | Unset = 1,

) -> Response[PartnerRequestLog | Problem]:
    """ A partner link's request log (90 days; never the addresses)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        link (UUID):
        page (int | Unset):  Default: 1.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[PartnerRequestLog | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
link=link,
page=page,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    link: UUID,
    *,
    client: AuthenticatedClient,
    page: int | Unset = 1,

) -> PartnerRequestLog | Problem | None:
    """ A partner link's request log (90 days; never the addresses)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        link (UUID):
        page (int | Unset):  Default: 1.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        PartnerRequestLog | Problem
     """


    return sync_detailed(
        workspace=workspace,
link=link,
client=client,
page=page,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    link: UUID,
    *,
    client: AuthenticatedClient,
    page: int | Unset = 1,

) -> Response[PartnerRequestLog | Problem]:
    """ A partner link's request log (90 days; never the addresses)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        link (UUID):
        page (int | Unset):  Default: 1.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[PartnerRequestLog | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
link=link,
page=page,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    link: UUID,
    *,
    client: AuthenticatedClient,
    page: int | Unset = 1,

) -> PartnerRequestLog | Problem | None:
    """ A partner link's request log (90 days; never the addresses)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        link (UUID):
        page (int | Unset):  Default: 1.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        PartnerRequestLog | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
link=link,
client=client,
page=page,

    )).parsed
