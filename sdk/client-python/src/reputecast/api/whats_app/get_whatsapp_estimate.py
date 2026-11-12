from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.whats_app_estimate import WhatsAppEstimate
from ...types import UNSET, Unset
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    *,
    list_: UUID,
    template: str | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    json_list_ = str(list_)
    params["list"] = json_list_

    params["template"] = template


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/whatsapp/estimate".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | WhatsAppEstimate:
    if response.status_code == 200:
        response_200 = WhatsAppEstimate.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | WhatsAppEstimate]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    list_: UUID,
    template: str | Unset = UNSET,

) -> Response[Problem | WhatsAppEstimate]:
    """ Who a list's WhatsApp consent reaches and what one template to them costs

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        list_ (UUID):
        template (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WhatsAppEstimate]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
list_=list_,
template=template,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    list_: UUID,
    template: str | Unset = UNSET,

) -> Problem | WhatsAppEstimate | None:
    """ Who a list's WhatsApp consent reaches and what one template to them costs

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        list_ (UUID):
        template (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WhatsAppEstimate
     """


    return sync_detailed(
        workspace=workspace,
client=client,
list_=list_,
template=template,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    list_: UUID,
    template: str | Unset = UNSET,

) -> Response[Problem | WhatsAppEstimate]:
    """ Who a list's WhatsApp consent reaches and what one template to them costs

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        list_ (UUID):
        template (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WhatsAppEstimate]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
list_=list_,
template=template,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    list_: UUID,
    template: str | Unset = UNSET,

) -> Problem | WhatsAppEstimate | None:
    """ Who a list's WhatsApp consent reaches and what one template to them costs

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        list_ (UUID):
        template (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WhatsAppEstimate
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
list_=list_,
template=template,

    )).parsed
