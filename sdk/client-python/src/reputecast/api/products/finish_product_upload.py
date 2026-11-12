from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.product_file import ProductFile
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    product: UUID,
    upload: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/products/{product}/uploads/{upload}/complete".format(workspace=quote(str(workspace), safe=""),product=quote(str(product), safe=""),upload=quote(str(upload), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | ProductFile:
    if response.status_code == 200:
        response_200 = ProductFile.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | ProductFile]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    product: UUID,
    upload: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | ProductFile]:
    """ Finish an upload: the file's type is checked from its content

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        product (UUID):
        upload (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ProductFile]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
product=product,
upload=upload,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    product: UUID,
    upload: UUID,
    *,
    client: AuthenticatedClient,

) -> Problem | ProductFile | None:
    """ Finish an upload: the file's type is checked from its content

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        product (UUID):
        upload (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ProductFile
     """


    return sync_detailed(
        workspace=workspace,
product=product,
upload=upload,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    product: UUID,
    upload: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | ProductFile]:
    """ Finish an upload: the file's type is checked from its content

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        product (UUID):
        upload (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ProductFile]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
product=product,
upload=upload,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    product: UUID,
    upload: UUID,
    *,
    client: AuthenticatedClient,

) -> Problem | ProductFile | None:
    """ Finish an upload: the file's type is checked from its content

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        product (UUID):
        upload (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ProductFile
     """


    return (await asyncio_detailed(
        workspace=workspace,
product=product,
upload=upload,
client=client,

    )).parsed
