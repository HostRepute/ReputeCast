from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.media_asset import MediaAsset
from ...models.problem import Problem
from ...models.upload_media_body import UploadMediaBody
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    body: UploadMediaBody | Unset = UNSET,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/media".format(workspace=quote(str(workspace), safe=""),),
    }

    if not isinstance(body, Unset):
        _kwargs["files"] = body.to_multipart()

    headers["Content-Type"] = "multipart/form-data; boundary=+++"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> MediaAsset | Problem:
    if response.status_code == 201:
        response_201 = MediaAsset.from_dict(response.json())



        return response_201

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[MediaAsset | Problem]:
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
    body: UploadMediaBody | Unset = UNSET,

) -> Response[MediaAsset | Problem]:
    """ Upload an image (PNG or JPEG up to 20 MB, GIF up to 5 MB)

     The file is validated by its content, not its name. What's stored is made for email: at most 1,200
    px wide (same proportions, never enlarged), photos turned upright and compressed (a photo saved as
    PNG may become a JPEG), and metadata such as location and camera details removed. GIFs keep their
    frames. `original` describes the file as uploaded when the stored image differs from it. Uploading
    the same file twice returns the stored image.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (UploadMediaBody | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[MediaAsset | Problem]
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
    body: UploadMediaBody | Unset = UNSET,

) -> MediaAsset | Problem | None:
    """ Upload an image (PNG or JPEG up to 20 MB, GIF up to 5 MB)

     The file is validated by its content, not its name. What's stored is made for email: at most 1,200
    px wide (same proportions, never enlarged), photos turned upright and compressed (a photo saved as
    PNG may become a JPEG), and metadata such as location and camera details removed. GIFs keep their
    frames. `original` describes the file as uploaded when the stored image differs from it. Uploading
    the same file twice returns the stored image.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (UploadMediaBody | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        MediaAsset | Problem
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
    body: UploadMediaBody | Unset = UNSET,

) -> Response[MediaAsset | Problem]:
    """ Upload an image (PNG or JPEG up to 20 MB, GIF up to 5 MB)

     The file is validated by its content, not its name. What's stored is made for email: at most 1,200
    px wide (same proportions, never enlarged), photos turned upright and compressed (a photo saved as
    PNG may become a JPEG), and metadata such as location and camera details removed. GIFs keep their
    frames. `original` describes the file as uploaded when the stored image differs from it. Uploading
    the same file twice returns the stored image.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (UploadMediaBody | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[MediaAsset | Problem]
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
    body: UploadMediaBody | Unset = UNSET,

) -> MediaAsset | Problem | None:
    """ Upload an image (PNG or JPEG up to 20 MB, GIF up to 5 MB)

     The file is validated by its content, not its name. What's stored is made for email: at most 1,200
    px wide (same proportions, never enlarged), photos turned upright and compressed (a photo saved as
    PNG may become a JPEG), and metadata such as location and camera details removed. GIFs keep their
    frames. `original` describes the file as uploaded when the stored image differs from it. Uploading
    the same file twice returns the stored image.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (UploadMediaBody | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        MediaAsset | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
body=body,

    )).parsed
