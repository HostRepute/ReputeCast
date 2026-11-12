from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.upload_state import UploadState
from ...types import File, FileTypes
from ...types import UNSET, Unset
from io import BytesIO
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    form: UUID,
    upload: UUID,
    *,
    body: File,
    offset: int | Unset = UNSET,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    params: dict[str, Any] = {}

    params["offset"] = offset


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/v1/workspaces/{workspace}/forms/{form}/download/uploads/{upload}".format(workspace=quote(str(workspace), safe=""),form=quote(str(form), safe=""),upload=quote(str(upload), safe=""),),
        "params": params,
    }

    _kwargs["content"] = body.payload
    headers["Content-Type"] = "application/octet-stream"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | UploadState:
    if response.status_code == 200:
        response_200 = UploadState.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | UploadState]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    form: UUID,
    upload: UUID,
    *,
    client: AuthenticatedClient,
    body: File,
    offset: int | Unset = UNSET,

) -> Response[Problem | UploadState]:
    """ Send the next chunk of the file

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (UUID): Form id
        upload (UUID):
        offset (int | Unset): Where this chunk starts: the bytes received so far
        body (File):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | UploadState]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
form=form,
upload=upload,
body=body,
offset=offset,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    form: UUID,
    upload: UUID,
    *,
    client: AuthenticatedClient,
    body: File,
    offset: int | Unset = UNSET,

) -> Problem | UploadState | None:
    """ Send the next chunk of the file

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (UUID): Form id
        upload (UUID):
        offset (int | Unset): Where this chunk starts: the bytes received so far
        body (File):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | UploadState
     """


    return sync_detailed(
        workspace=workspace,
form=form,
upload=upload,
client=client,
body=body,
offset=offset,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    form: UUID,
    upload: UUID,
    *,
    client: AuthenticatedClient,
    body: File,
    offset: int | Unset = UNSET,

) -> Response[Problem | UploadState]:
    """ Send the next chunk of the file

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (UUID): Form id
        upload (UUID):
        offset (int | Unset): Where this chunk starts: the bytes received so far
        body (File):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | UploadState]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
form=form,
upload=upload,
body=body,
offset=offset,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    form: UUID,
    upload: UUID,
    *,
    client: AuthenticatedClient,
    body: File,
    offset: int | Unset = UNSET,

) -> Problem | UploadState | None:
    """ Send the next chunk of the file

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (UUID): Form id
        upload (UUID):
        offset (int | Unset): Where this chunk starts: the bytes received so far
        body (File):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | UploadState
     """


    return (await asyncio_detailed(
        workspace=workspace,
form=form,
upload=upload,
client=client,
body=body,
offset=offset,

    )).parsed
