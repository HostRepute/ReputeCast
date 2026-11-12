from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.product_detail import ProductDetail
from ...models.product_publish_input import ProductPublishInput
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    product: UUID,
    *,
    body: ProductPublishInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/products/{product}/publish".format(workspace=quote(str(workspace), safe=""),product=quote(str(product), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | ProductDetail:
    if response.status_code == 200:
        response_200 = ProductDetail.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | ProductDetail]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    product: UUID,
    *,
    client: AuthenticatedClient,
    body: ProductPublishInput,

) -> Response[Problem | ProductDetail]:
    """ Publish the draft: its price in Stripe, the shop and its pages

     Blocked until every publish check passes. A new price on a published product needs confirmPrice:
    existing buyers keep their access.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        product (UUID):
        body (ProductPublishInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ProductDetail]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
product=product,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    product: UUID,
    *,
    client: AuthenticatedClient,
    body: ProductPublishInput,

) -> Problem | ProductDetail | None:
    """ Publish the draft: its price in Stripe, the shop and its pages

     Blocked until every publish check passes. A new price on a published product needs confirmPrice:
    existing buyers keep their access.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        product (UUID):
        body (ProductPublishInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ProductDetail
     """


    return sync_detailed(
        workspace=workspace,
product=product,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    product: UUID,
    *,
    client: AuthenticatedClient,
    body: ProductPublishInput,

) -> Response[Problem | ProductDetail]:
    """ Publish the draft: its price in Stripe, the shop and its pages

     Blocked until every publish check passes. A new price on a published product needs confirmPrice:
    existing buyers keep their access.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        product (UUID):
        body (ProductPublishInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ProductDetail]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
product=product,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    product: UUID,
    *,
    client: AuthenticatedClient,
    body: ProductPublishInput,

) -> Problem | ProductDetail | None:
    """ Publish the draft: its price in Stripe, the shop and its pages

     Blocked until every publish check passes. A new price on a published product needs confirmPrice:
    existing buyers keep their access.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        product (UUID):
        body (ProductPublishInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ProductDetail
     """


    return (await asyncio_detailed(
        workspace=workspace,
product=product,
client=client,
body=body,

    )).parsed
