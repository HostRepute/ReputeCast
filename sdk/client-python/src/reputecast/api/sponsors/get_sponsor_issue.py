from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.sponsor_issue_detail import SponsorIssueDetail
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    id: UUID,
    number: int,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/sponsor-slots/{id}/issues/{number}".format(workspace=quote(str(workspace), safe=""),id=quote(str(id), safe=""),number=quote(str(number), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | SponsorIssueDetail:
    if response.status_code == 200:
        response_200 = SponsorIssueDetail.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | SponsorIssueDetail]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    id: UUID,
    number: int,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | SponsorIssueDetail]:
    """ An issue of a slot: its dates, price, campaign and any proposal

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID): The slot's id
        number (int): The issue's number

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SponsorIssueDetail]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
number=number,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    id: UUID,
    number: int,
    *,
    client: AuthenticatedClient,

) -> Problem | SponsorIssueDetail | None:
    """ An issue of a slot: its dates, price, campaign and any proposal

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID): The slot's id
        number (int): The issue's number

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SponsorIssueDetail
     """


    return sync_detailed(
        workspace=workspace,
id=id,
number=number,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    id: UUID,
    number: int,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | SponsorIssueDetail]:
    """ An issue of a slot: its dates, price, campaign and any proposal

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID): The slot's id
        number (int): The issue's number

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SponsorIssueDetail]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
number=number,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    id: UUID,
    number: int,
    *,
    client: AuthenticatedClient,

) -> Problem | SponsorIssueDetail | None:
    """ An issue of a slot: its dates, price, campaign and any proposal

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID): The slot's id
        number (int): The issue's number

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SponsorIssueDetail
     """


    return (await asyncio_detailed(
        workspace=workspace,
id=id,
number=number,
client=client,

    )).parsed
