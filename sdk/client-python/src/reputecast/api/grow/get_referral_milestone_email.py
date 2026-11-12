from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.referral_milestone_email_detail import ReferralMilestoneEmailDetail
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    milestone: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/grow/milestones/{milestone}/email".format(workspace=quote(str(workspace), safe=""),milestone=quote(str(milestone), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | ReferralMilestoneEmailDetail:
    if response.status_code == 200:
        response_200 = ReferralMilestoneEmailDetail.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | ReferralMilestoneEmailDetail]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    milestone: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | ReferralMilestoneEmailDetail]:
    """ A milestone email for the composer's milestone mode

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        milestone (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ReferralMilestoneEmailDetail]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
milestone=milestone,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    milestone: UUID,
    *,
    client: AuthenticatedClient,

) -> Problem | ReferralMilestoneEmailDetail | None:
    """ A milestone email for the composer's milestone mode

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        milestone (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ReferralMilestoneEmailDetail
     """


    return sync_detailed(
        workspace=workspace,
milestone=milestone,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    milestone: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | ReferralMilestoneEmailDetail]:
    """ A milestone email for the composer's milestone mode

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        milestone (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ReferralMilestoneEmailDetail]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
milestone=milestone,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    milestone: UUID,
    *,
    client: AuthenticatedClient,

) -> Problem | ReferralMilestoneEmailDetail | None:
    """ A milestone email for the composer's milestone mode

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        milestone (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ReferralMilestoneEmailDetail
     """


    return (await asyncio_detailed(
        workspace=workspace,
milestone=milestone,
client=client,

    )).parsed
