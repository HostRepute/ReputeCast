from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.download_import_report_kind import DownloadImportReportKind
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    id: UUID,
    kind: DownloadImportReportKind,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/imports/{id}/files/{kind}".format(workspace=quote(str(workspace), safe=""),id=quote(str(id), safe=""),kind=quote(str(kind), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | str:
    if response.status_code == 200:
        response_200 = cast(str, response.json())
        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | str]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    id: UUID,
    kind: DownloadImportReportKind,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | str]:
    """ Download the full report, skipped rows, invalid rows or unreadable file rows (logged)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        kind (DownloadImportReportKind):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | str]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
kind=kind,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    id: UUID,
    kind: DownloadImportReportKind,
    *,
    client: AuthenticatedClient,

) -> Problem | str | None:
    """ Download the full report, skipped rows, invalid rows or unreadable file rows (logged)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        kind (DownloadImportReportKind):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | str
     """


    return sync_detailed(
        workspace=workspace,
id=id,
kind=kind,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    id: UUID,
    kind: DownloadImportReportKind,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | str]:
    """ Download the full report, skipped rows, invalid rows or unreadable file rows (logged)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        kind (DownloadImportReportKind):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | str]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
kind=kind,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    id: UUID,
    kind: DownloadImportReportKind,
    *,
    client: AuthenticatedClient,

) -> Problem | str | None:
    """ Download the full report, skipped rows, invalid rows or unreadable file rows (logged)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        kind (DownloadImportReportKind):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | str
     """


    return (await asyncio_detailed(
        workspace=workspace,
id=id,
kind=kind,
client=client,

    )).parsed
