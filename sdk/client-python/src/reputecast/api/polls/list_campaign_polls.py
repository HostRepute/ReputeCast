from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.poll_results_list import PollResultsList
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    campaign_id: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/campaigns/{campaign_id}/polls".format(workspace=quote(str(workspace), safe=""),campaign_id=quote(str(campaign_id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> PollResultsList | Problem:
    if response.status_code == 200:
        response_200 = PollResultsList.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[PollResultsList | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    campaign_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[PollResultsList | Problem]:
    """ Results of a campaign's polls: answers by answer, NPS, breakdowns, the first 72 hours

     Answers are recorded on the answer page, never on the click: automated visits, answers from people
    who aren't contacts and test answers are counted apart.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID): Campaign id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[PollResultsList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
campaign_id=campaign_id,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    campaign_id: UUID,
    *,
    client: AuthenticatedClient,

) -> PollResultsList | Problem | None:
    """ Results of a campaign's polls: answers by answer, NPS, breakdowns, the first 72 hours

     Answers are recorded on the answer page, never on the click: automated visits, answers from people
    who aren't contacts and test answers are counted apart.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID): Campaign id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        PollResultsList | Problem
     """


    return sync_detailed(
        workspace=workspace,
campaign_id=campaign_id,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    campaign_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[PollResultsList | Problem]:
    """ Results of a campaign's polls: answers by answer, NPS, breakdowns, the first 72 hours

     Answers are recorded on the answer page, never on the click: automated visits, answers from people
    who aren't contacts and test answers are counted apart.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID): Campaign id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[PollResultsList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
campaign_id=campaign_id,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    campaign_id: UUID,
    *,
    client: AuthenticatedClient,

) -> PollResultsList | Problem | None:
    """ Results of a campaign's polls: answers by answer, NPS, breakdowns, the first 72 hours

     Answers are recorded on the answer page, never on the click: automated visits, answers from people
    who aren't contacts and test answers are counted apart.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID): Campaign id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        PollResultsList | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
campaign_id=campaign_id,
client=client,

    )).parsed
