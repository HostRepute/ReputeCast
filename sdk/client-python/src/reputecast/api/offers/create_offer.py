from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.offer_detail import OfferDetail
from ...models.offer_input import OfferInput
from ...models.problem import Problem
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    body: OfferInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/earn/offers".format(workspace=quote(str(workspace), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> OfferDetail | Problem:
    if response.status_code == 201:
        response_201 = OfferDetail.from_dict(response.json())



        return response_201

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[OfferDetail | Problem]:
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
    body: OfferInput,

) -> Response[OfferDetail | Problem]:
    """ Create an offer, and its coupon and codes in the payment provider

     Without a connected provider the offer is saved as a draft and created there once one is connected
    and it's saved again.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (OfferInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[OfferDetail | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: OfferInput,

) -> OfferDetail | Problem | None:
    """ Create an offer, and its coupon and codes in the payment provider

     Without a connected provider the offer is saved as a draft and created there once one is connected
    and it's saved again.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (OfferInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        OfferDetail | Problem
     """


    return sync_detailed(
        workspace=workspace,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: OfferInput,

) -> Response[OfferDetail | Problem]:
    """ Create an offer, and its coupon and codes in the payment provider

     Without a connected provider the offer is saved as a draft and created there once one is connected
    and it's saved again.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (OfferInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[OfferDetail | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: OfferInput,

) -> OfferDetail | Problem | None:
    """ Create an offer, and its coupon and codes in the payment provider

     Without a connected provider the offer is saved as a draft and created there once one is connected
    and it's saved again.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (OfferInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        OfferDetail | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
body=body,

    )).parsed
