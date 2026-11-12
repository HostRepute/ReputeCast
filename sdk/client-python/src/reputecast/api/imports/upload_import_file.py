from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.import_detail import ImportDetail
from ...models.problem import Problem
from ...types import File, FileTypes
from ...types import UNSET, Unset
from io import BytesIO
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    id: UUID,
    *,
    body: File,
    file_name: str | Unset = UNSET,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    params: dict[str, Any] = {}

    params["fileName"] = file_name


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/imports/{id}/file".format(workspace=quote(str(workspace), safe=""),id=quote(str(id), safe=""),),
        "params": params,
    }

    _kwargs["content"] = body.payload
    headers["Content-Type"] = "application/octet-stream"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> ImportDetail | Problem:
    if response.status_code == 200:
        response_200 = ImportDetail.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[ImportDetail | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    id: UUID,
    *,
    client: AuthenticatedClient,
    body: File,
    file_name: str | Unset = UNSET,

) -> Response[ImportDetail | Problem]:
    """ Upload a CSV or XLSX file (first row: headers)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        file_name (str | Unset): The file's name
        body (File):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ImportDetail | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
body=body,
file_name=file_name,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    id: UUID,
    *,
    client: AuthenticatedClient,
    body: File,
    file_name: str | Unset = UNSET,

) -> ImportDetail | Problem | None:
    """ Upload a CSV or XLSX file (first row: headers)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        file_name (str | Unset): The file's name
        body (File):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ImportDetail | Problem
     """


    return sync_detailed(
        workspace=workspace,
id=id,
client=client,
body=body,
file_name=file_name,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    id: UUID,
    *,
    client: AuthenticatedClient,
    body: File,
    file_name: str | Unset = UNSET,

) -> Response[ImportDetail | Problem]:
    """ Upload a CSV or XLSX file (first row: headers)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        file_name (str | Unset): The file's name
        body (File):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ImportDetail | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
body=body,
file_name=file_name,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    id: UUID,
    *,
    client: AuthenticatedClient,
    body: File,
    file_name: str | Unset = UNSET,

) -> ImportDetail | Problem | None:
    """ Upload a CSV or XLSX file (first row: headers)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        file_name (str | Unset): The file's name
        body (File):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ImportDetail | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
id=id,
client=client,
body=body,
file_name=file_name,

    )).parsed
