from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.referral_referrer_panel import ReferralReferrerPanel
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    program: UUID,
    link: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/grow/referrals/{program}/referrers/{link}".format(workspace=quote(str(workspace), safe=""),program=quote(str(program), safe=""),link=quote(str(link), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | ReferralReferrerPanel:
    if response.status_code == 200:
        response_200 = ReferralReferrerPanel.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | ReferralReferrerPanel]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    program: UUID,
    link: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | ReferralReferrerPanel]:
    """ A referrer: their link, progress, referrals and rewards

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        program (UUID):
        link (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ReferralReferrerPanel]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
program=program,
link=link,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    program: UUID,
    link: UUID,
    *,
    client: AuthenticatedClient,

) -> Problem | ReferralReferrerPanel | None:
    """ A referrer: their link, progress, referrals and rewards

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        program (UUID):
        link (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ReferralReferrerPanel
     """


    return sync_detailed(
        workspace=workspace,
program=program,
link=link,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    program: UUID,
    link: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | ReferralReferrerPanel]:
    """ A referrer: their link, progress, referrals and rewards

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        program (UUID):
        link (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ReferralReferrerPanel]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
program=program,
link=link,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    program: UUID,
    link: UUID,
    *,
    client: AuthenticatedClient,

) -> Problem | ReferralReferrerPanel | None:
    """ A referrer: their link, progress, referrals and rewards

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        program (UUID):
        link (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ReferralReferrerPanel
     """


    return (await asyncio_detailed(
        workspace=workspace,
program=program,
link=link,
client=client,

    )).parsed
