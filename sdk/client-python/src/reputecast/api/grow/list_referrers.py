from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.referral_referrer_list import ReferralReferrerList
from ...types import UNSET, Unset
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    program: UUID,
    *,
    q: str | Unset = UNSET,
    milestone: int | Unset = UNSET,
    held: bool | Unset = UNSET,
    paused: bool | Unset = UNSET,
    page: int | Unset = 1,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["q"] = q

    params["milestone"] = milestone

    params["held"] = held

    params["paused"] = paused

    params["page"] = page


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/grow/referrals/{program}/referrers".format(workspace=quote(str(workspace), safe=""),program=quote(str(program), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | ReferralReferrerList:
    if response.status_code == 200:
        response_200 = ReferralReferrerList.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | ReferralReferrerList]:
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
    q: str | Unset = UNSET,
    milestone: int | Unset = UNSET,
    held: bool | Unset = UNSET,
    paused: bool | Unset = UNSET,
    page: int | Unset = 1,

) -> Response[Problem | ReferralReferrerList]:
    """ Referrers by counted referrals, with search and filters

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        program (UUID):
        q (str | Unset):
        milestone (int | Unset): 0 any, -1 none yet, n reached n
        held (bool | Unset):
        paused (bool | Unset):
        page (int | Unset):  Default: 1.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ReferralReferrerList]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
program=program,
q=q,
milestone=milestone,
held=held,
paused=paused,
page=page,

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
    q: str | Unset = UNSET,
    milestone: int | Unset = UNSET,
    held: bool | Unset = UNSET,
    paused: bool | Unset = UNSET,
    page: int | Unset = 1,

) -> Problem | ReferralReferrerList | None:
    """ Referrers by counted referrals, with search and filters

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        program (UUID):
        q (str | Unset):
        milestone (int | Unset): 0 any, -1 none yet, n reached n
        held (bool | Unset):
        paused (bool | Unset):
        page (int | Unset):  Default: 1.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ReferralReferrerList
     """


    return sync_detailed(
        workspace=workspace,
program=program,
client=client,
q=q,
milestone=milestone,
held=held,
paused=paused,
page=page,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    program: UUID,
    *,
    client: AuthenticatedClient,
    q: str | Unset = UNSET,
    milestone: int | Unset = UNSET,
    held: bool | Unset = UNSET,
    paused: bool | Unset = UNSET,
    page: int | Unset = 1,

) -> Response[Problem | ReferralReferrerList]:
    """ Referrers by counted referrals, with search and filters

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        program (UUID):
        q (str | Unset):
        milestone (int | Unset): 0 any, -1 none yet, n reached n
        held (bool | Unset):
        paused (bool | Unset):
        page (int | Unset):  Default: 1.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ReferralReferrerList]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
program=program,
q=q,
milestone=milestone,
held=held,
paused=paused,
page=page,

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
    q: str | Unset = UNSET,
    milestone: int | Unset = UNSET,
    held: bool | Unset = UNSET,
    paused: bool | Unset = UNSET,
    page: int | Unset = 1,

) -> Problem | ReferralReferrerList | None:
    """ Referrers by counted referrals, with search and filters

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        program (UUID):
        q (str | Unset):
        milestone (int | Unset): 0 any, -1 none yet, n reached n
        held (bool | Unset):
        paused (bool | Unset):
        page (int | Unset):  Default: 1.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ReferralReferrerList
     """


    return (await asyncio_detailed(
        workspace=workspace,
program=program,
client=client,
q=q,
milestone=milestone,
held=held,
paused=paused,
page=page,

    )).parsed
