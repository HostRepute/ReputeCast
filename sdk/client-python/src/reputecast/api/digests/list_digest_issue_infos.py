from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.digest_issue_infos import DigestIssueInfos
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    campaigns: str | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["campaigns"] = campaigns


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/digest-issues".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> DigestIssueInfos | Problem:
    if response.status_code == 200:
        response_200 = DigestIssueInfos.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[DigestIssueInfos | Problem]:
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
    campaigns: str | Unset = UNSET,

) -> Response[DigestIssueInfos | Problem]:
    """ Which campaigns are digest issues, with their posts and what changed since the last approved issue

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaigns (str | Unset): Campaign ids, comma-separated (up to 100)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[DigestIssueInfos | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
campaigns=campaigns,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    campaigns: str | Unset = UNSET,

) -> DigestIssueInfos | Problem | None:
    """ Which campaigns are digest issues, with their posts and what changed since the last approved issue

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaigns (str | Unset): Campaign ids, comma-separated (up to 100)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        DigestIssueInfos | Problem
     """


    return sync_detailed(
        workspace=workspace,
client=client,
campaigns=campaigns,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    campaigns: str | Unset = UNSET,

) -> Response[DigestIssueInfos | Problem]:
    """ Which campaigns are digest issues, with their posts and what changed since the last approved issue

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaigns (str | Unset): Campaign ids, comma-separated (up to 100)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[DigestIssueInfos | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
campaigns=campaigns,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    campaigns: str | Unset = UNSET,

) -> DigestIssueInfos | Problem | None:
    """ Which campaigns are digest issues, with their posts and what changed since the last approved issue

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaigns (str | Unset): Campaign ids, comma-separated (up to 100)

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        DigestIssueInfos | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
campaigns=campaigns,

    )).parsed
