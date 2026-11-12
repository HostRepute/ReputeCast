from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.product_cover import ProductCover
from ...models.upload_product_cover_body import UploadProductCoverBody
from ...types import UNSET, Unset
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    product: UUID,
    *,
    body: UploadProductCoverBody | Unset = UNSET,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/products/{product}/cover".format(workspace=quote(str(workspace), safe=""),product=quote(str(product), safe=""),),
    }

    if not isinstance(body, Unset):
        _kwargs["files"] = body.to_multipart()

    headers["Content-Type"] = "multipart/form-data; boundary=+++"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | ProductCover:
    if response.status_code == 201:
        response_201 = ProductCover.from_dict(response.json())



        return response_201

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | ProductCover]:
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
    body: UploadProductCoverBody | Unset = UNSET,

) -> Response[Problem | ProductCover]:
    """ Upload a cover image (PNG, JPEG or GIF, up to 5 MB)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        product (UUID):
        body (UploadProductCoverBody | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ProductCover]
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
    body: UploadProductCoverBody | Unset = UNSET,

) -> Problem | ProductCover | None:
    """ Upload a cover image (PNG, JPEG or GIF, up to 5 MB)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        product (UUID):
        body (UploadProductCoverBody | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ProductCover
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
    body: UploadProductCoverBody | Unset = UNSET,

) -> Response[Problem | ProductCover]:
    """ Upload a cover image (PNG, JPEG or GIF, up to 5 MB)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        product (UUID):
        body (UploadProductCoverBody | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ProductCover]
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
    body: UploadProductCoverBody | Unset = UNSET,

) -> Problem | ProductCover | None:
    """ Upload a cover image (PNG, JPEG or GIF, up to 5 MB)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        product (UUID):
        body (UploadProductCoverBody | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ProductCover
     """


    return (await asyncio_detailed(
        workspace=workspace,
product=product,
client=client,
body=body,

    )).parsed
