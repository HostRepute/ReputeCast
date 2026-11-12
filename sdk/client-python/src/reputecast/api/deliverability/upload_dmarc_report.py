from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.deliverability_dmarc_upload import DeliverabilityDMARCUpload
from ...models.problem import Problem
from ...types import File, FileTypes
from io import BytesIO
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    body: File,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/deliverability/dmarc/reports".format(workspace=quote(str(workspace), safe=""),),
    }

    _kwargs["content"] = body.payload
    headers["Content-Type"] = "application/octet-stream"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> DeliverabilityDMARCUpload | Problem:
    if response.status_code == 201:
        response_201 = DeliverabilityDMARCUpload.from_dict(response.json())



        return response_201

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[DeliverabilityDMARCUpload | Problem]:
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
    body: File,

) -> Response[DeliverabilityDMARCUpload | Problem]:
    """ Upload a DMARC aggregate report (XML, .xml.gz or .zip)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (File):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[DeliverabilityDMARCUpload | Problem]
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
    body: File,

) -> DeliverabilityDMARCUpload | Problem | None:
    """ Upload a DMARC aggregate report (XML, .xml.gz or .zip)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (File):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        DeliverabilityDMARCUpload | Problem
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
    body: File,

) -> Response[DeliverabilityDMARCUpload | Problem]:
    """ Upload a DMARC aggregate report (XML, .xml.gz or .zip)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (File):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[DeliverabilityDMARCUpload | Problem]
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
    body: File,

) -> DeliverabilityDMARCUpload | Problem | None:
    """ Upload a DMARC aggregate report (XML, .xml.gz or .zip)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (File):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        DeliverabilityDMARCUpload | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
body=body,

    )).parsed
