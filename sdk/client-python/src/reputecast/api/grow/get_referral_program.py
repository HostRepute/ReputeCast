from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.referral_program_detail import ReferralProgramDetail
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    program: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/grow/referrals/{program}".format(workspace=quote(str(workspace), safe=""),program=quote(str(program), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | ReferralProgramDetail:
    if response.status_code == 200:
        response_200 = ReferralProgramDetail.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | ReferralProgramDetail]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    program: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | ReferralProgramDetail]:
    """ A referral program with its settings

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        program (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ReferralProgramDetail]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
program=program,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    program: UUID,
    *,
    client: AuthenticatedClient,

) -> Problem | ReferralProgramDetail | None:
    """ A referral program with its settings

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        program (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ReferralProgramDetail
     """


    return sync_detailed(
        workspace=workspace,
program=program,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    program: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | ReferralProgramDetail]:
    """ A referral program with its settings

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        program (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ReferralProgramDetail]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
program=program,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    program: UUID,
    *,
    client: AuthenticatedClient,

) -> Problem | ReferralProgramDetail | None:
    """ A referral program with its settings

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        program (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ReferralProgramDetail
     """


    return (await asyncio_detailed(
        workspace=workspace,
program=program,
client=client,

    )).parsed
