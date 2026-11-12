from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.approval_preview import ApprovalPreview
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    approval_id: UUID,
    *,
    language: str | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["language"] = language


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/approvals/{approval_id}/preview".format(workspace=quote(str(workspace), safe=""),approval_id=quote(str(approval_id), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> ApprovalPreview | Problem:
    if response.status_code == 200:
        response_200 = ApprovalPreview.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[ApprovalPreview | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    approval_id: UUID,
    *,
    client: AuthenticatedClient,
    language: str | Unset = UNSET,

) -> Response[ApprovalPreview | Problem]:
    """ The email an approval covers

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        approval_id (UUID):
        language (str | Unset): Preview a language version the approval covers; empty shows the
            source

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ApprovalPreview | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
approval_id=approval_id,
language=language,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    approval_id: UUID,
    *,
    client: AuthenticatedClient,
    language: str | Unset = UNSET,

) -> ApprovalPreview | Problem | None:
    """ The email an approval covers

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        approval_id (UUID):
        language (str | Unset): Preview a language version the approval covers; empty shows the
            source

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ApprovalPreview | Problem
     """


    return sync_detailed(
        workspace=workspace,
approval_id=approval_id,
client=client,
language=language,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    approval_id: UUID,
    *,
    client: AuthenticatedClient,
    language: str | Unset = UNSET,

) -> Response[ApprovalPreview | Problem]:
    """ The email an approval covers

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        approval_id (UUID):
        language (str | Unset): Preview a language version the approval covers; empty shows the
            source

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ApprovalPreview | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
approval_id=approval_id,
language=language,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    approval_id: UUID,
    *,
    client: AuthenticatedClient,
    language: str | Unset = UNSET,

) -> ApprovalPreview | Problem | None:
    """ The email an approval covers

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        approval_id (UUID):
        language (str | Unset): Preview a language version the approval covers; empty shows the
            source

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ApprovalPreview | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
approval_id=approval_id,
client=client,
language=language,

    )).parsed
