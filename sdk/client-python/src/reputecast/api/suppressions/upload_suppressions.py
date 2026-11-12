from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.suppression_upload_result import SuppressionUploadResult
from ...models.upload_suppressions_reason import UploadSuppressionsReason
from ...types import File, FileTypes
from ...types import UNSET, Unset
from io import BytesIO
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    body: File,
    file_name: str | Unset = UNSET,
    reason: UploadSuppressionsReason | Unset = UploadSuppressionsReason.MANUAL,
    note: str | Unset = UNSET,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    params: dict[str, Any] = {}

    params["fileName"] = file_name

    json_reason: str | Unset = UNSET
    if not isinstance(reason, Unset):
        json_reason = reason.value

    params["reason"] = json_reason

    params["note"] = note


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/suppressions/upload".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }

    _kwargs["content"] = body.payload
    headers["Content-Type"] = "text/csv"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | SuppressionUploadResult:
    if response.status_code == 200:
        response_200 = SuppressionUploadResult.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | SuppressionUploadResult]:
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
    file_name: str | Unset = UNSET,
    reason: UploadSuppressionsReason | Unset = UploadSuppressionsReason.MANUAL,
    note: str | Unset = UNSET,

) -> Response[Problem | SuppressionUploadResult]:
    """ Add addresses from a CSV file

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        file_name (str | Unset):
        reason (UploadSuppressionsReason | Unset):  Default: UploadSuppressionsReason.MANUAL.
        note (str | Unset):
        body (File):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SuppressionUploadResult]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
body=body,
file_name=file_name,
reason=reason,
note=note,

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
    file_name: str | Unset = UNSET,
    reason: UploadSuppressionsReason | Unset = UploadSuppressionsReason.MANUAL,
    note: str | Unset = UNSET,

) -> Problem | SuppressionUploadResult | None:
    """ Add addresses from a CSV file

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        file_name (str | Unset):
        reason (UploadSuppressionsReason | Unset):  Default: UploadSuppressionsReason.MANUAL.
        note (str | Unset):
        body (File):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SuppressionUploadResult
     """


    return sync_detailed(
        workspace=workspace,
client=client,
body=body,
file_name=file_name,
reason=reason,
note=note,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: File,
    file_name: str | Unset = UNSET,
    reason: UploadSuppressionsReason | Unset = UploadSuppressionsReason.MANUAL,
    note: str | Unset = UNSET,

) -> Response[Problem | SuppressionUploadResult]:
    """ Add addresses from a CSV file

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        file_name (str | Unset):
        reason (UploadSuppressionsReason | Unset):  Default: UploadSuppressionsReason.MANUAL.
        note (str | Unset):
        body (File):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SuppressionUploadResult]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
body=body,
file_name=file_name,
reason=reason,
note=note,

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
    file_name: str | Unset = UNSET,
    reason: UploadSuppressionsReason | Unset = UploadSuppressionsReason.MANUAL,
    note: str | Unset = UNSET,

) -> Problem | SuppressionUploadResult | None:
    """ Add addresses from a CSV file

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        file_name (str | Unset):
        reason (UploadSuppressionsReason | Unset):  Default: UploadSuppressionsReason.MANUAL.
        note (str | Unset):
        body (File):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SuppressionUploadResult
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
body=body,
file_name=file_name,
reason=reason,
note=note,

    )).parsed
