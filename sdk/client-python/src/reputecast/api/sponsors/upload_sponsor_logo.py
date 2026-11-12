from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.sponsor_image import SponsorImage
from ...models.upload_sponsor_logo_body import UploadSponsorLogoBody
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    body: UploadSponsorLogoBody | Unset = UNSET,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/sponsors/logo".format(workspace=quote(str(workspace), safe=""),),
    }

    if not isinstance(body, Unset):
        _kwargs["files"] = body.to_multipart()

    headers["Content-Type"] = "multipart/form-data; boundary=+++"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | SponsorImage:
    if response.status_code == 201:
        response_201 = SponsorImage.from_dict(response.json())



        return response_201

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | SponsorImage]:
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
    body: UploadSponsorLogoBody | Unset = UNSET,

) -> Response[Problem | SponsorImage]:
    """ Upload a sponsor's logo (PNG, JPEG or GIF, up to 5 MB)

     The logo is kept in the media library; save its URL as the sponsor's logoUrl.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (UploadSponsorLogoBody | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SponsorImage]
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
    body: UploadSponsorLogoBody | Unset = UNSET,

) -> Problem | SponsorImage | None:
    """ Upload a sponsor's logo (PNG, JPEG or GIF, up to 5 MB)

     The logo is kept in the media library; save its URL as the sponsor's logoUrl.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (UploadSponsorLogoBody | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SponsorImage
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
    body: UploadSponsorLogoBody | Unset = UNSET,

) -> Response[Problem | SponsorImage]:
    """ Upload a sponsor's logo (PNG, JPEG or GIF, up to 5 MB)

     The logo is kept in the media library; save its URL as the sponsor's logoUrl.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (UploadSponsorLogoBody | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SponsorImage]
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
    body: UploadSponsorLogoBody | Unset = UNSET,

) -> Problem | SponsorImage | None:
    """ Upload a sponsor's logo (PNG, JPEG or GIF, up to 5 MB)

     The logo is kept in the media library; save its URL as the sponsor's logoUrl.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (UploadSponsorLogoBody | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SponsorImage
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
body=body,

    )).parsed
