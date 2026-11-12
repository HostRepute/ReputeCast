from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.approval_list import ApprovalList
from ...models.list_approvals_status import ListApprovalsStatus
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    status: ListApprovalsStatus | Unset = ListApprovalsStatus.PENDING,
    author: str | Unset = UNSET,
    approver: str | Unset = UNSET,
    days: int | Unset = 30,
    q: str | Unset = UNSET,
    cursor: str | Unset = UNSET,
    limit: int | Unset = 50,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    json_status: str | Unset = UNSET
    if not isinstance(status, Unset):
        json_status = status.value

    params["status"] = json_status

    params["author"] = author

    params["approver"] = approver

    params["days"] = days

    params["q"] = q

    params["cursor"] = cursor

    params["limit"] = limit


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/approvals".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> ApprovalList | Problem:
    if response.status_code == 200:
        response_200 = ApprovalList.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[ApprovalList | Problem]:
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
    status: ListApprovalsStatus | Unset = ListApprovalsStatus.PENDING,
    author: str | Unset = UNSET,
    approver: str | Unset = UNSET,
    days: int | Unset = 30,
    q: str | Unset = UNSET,
    cursor: str | Unset = UNSET,
    limit: int | Unset = 50,

) -> Response[ApprovalList | Problem]:
    """ The approval inbox: approvers see every request, authors their own

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ListApprovalsStatus | Unset):  Default: ListApprovalsStatus.PENDING.
        author (str | Unset): Requester user id
        approver (str | Unset): Approver user id
        days (int | Unset): Requested in the last N days (0: all) Default: 30.
        q (str | Unset): Search the campaigns, flows, templates and pages asked about by name
        cursor (str | Unset): The nextCursor of the page before
        limit (int | Unset):  Default: 50.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ApprovalList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
status=status,
author=author,
approver=approver,
days=days,
q=q,
cursor=cursor,
limit=limit,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    status: ListApprovalsStatus | Unset = ListApprovalsStatus.PENDING,
    author: str | Unset = UNSET,
    approver: str | Unset = UNSET,
    days: int | Unset = 30,
    q: str | Unset = UNSET,
    cursor: str | Unset = UNSET,
    limit: int | Unset = 50,

) -> ApprovalList | Problem | None:
    """ The approval inbox: approvers see every request, authors their own

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ListApprovalsStatus | Unset):  Default: ListApprovalsStatus.PENDING.
        author (str | Unset): Requester user id
        approver (str | Unset): Approver user id
        days (int | Unset): Requested in the last N days (0: all) Default: 30.
        q (str | Unset): Search the campaigns, flows, templates and pages asked about by name
        cursor (str | Unset): The nextCursor of the page before
        limit (int | Unset):  Default: 50.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ApprovalList | Problem
     """


    return sync_detailed(
        workspace=workspace,
client=client,
status=status,
author=author,
approver=approver,
days=days,
q=q,
cursor=cursor,
limit=limit,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    status: ListApprovalsStatus | Unset = ListApprovalsStatus.PENDING,
    author: str | Unset = UNSET,
    approver: str | Unset = UNSET,
    days: int | Unset = 30,
    q: str | Unset = UNSET,
    cursor: str | Unset = UNSET,
    limit: int | Unset = 50,

) -> Response[ApprovalList | Problem]:
    """ The approval inbox: approvers see every request, authors their own

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ListApprovalsStatus | Unset):  Default: ListApprovalsStatus.PENDING.
        author (str | Unset): Requester user id
        approver (str | Unset): Approver user id
        days (int | Unset): Requested in the last N days (0: all) Default: 30.
        q (str | Unset): Search the campaigns, flows, templates and pages asked about by name
        cursor (str | Unset): The nextCursor of the page before
        limit (int | Unset):  Default: 50.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ApprovalList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
status=status,
author=author,
approver=approver,
days=days,
q=q,
cursor=cursor,
limit=limit,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    status: ListApprovalsStatus | Unset = ListApprovalsStatus.PENDING,
    author: str | Unset = UNSET,
    approver: str | Unset = UNSET,
    days: int | Unset = 30,
    q: str | Unset = UNSET,
    cursor: str | Unset = UNSET,
    limit: int | Unset = 50,

) -> ApprovalList | Problem | None:
    """ The approval inbox: approvers see every request, authors their own

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        status (ListApprovalsStatus | Unset):  Default: ListApprovalsStatus.PENDING.
        author (str | Unset): Requester user id
        approver (str | Unset): Approver user id
        days (int | Unset): Requested in the last N days (0: all) Default: 30.
        q (str | Unset): Search the campaigns, flows, templates and pages asked about by name
        cursor (str | Unset): The nextCursor of the page before
        limit (int | Unset):  Default: 50.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ApprovalList | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
status=status,
author=author,
approver=approver,
days=days,
q=q,
cursor=cursor,
limit=limit,

    )).parsed
