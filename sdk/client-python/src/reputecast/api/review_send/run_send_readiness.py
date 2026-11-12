from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.readiness_poll import ReadinessPoll
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    campaign_id: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/campaigns/{campaign_id}/readiness".format(workspace=quote(str(workspace), safe=""),campaign_id=quote(str(campaign_id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | ReadinessPoll:
    if response.status_code == 202:
        response_202 = ReadinessPoll.from_dict(response.json())



        return response_202

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | ReadinessPoll]:
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

) -> Response[Problem | ReadinessPoll]:
    """ Run Send Readiness on a campaign

     Queues a run of every check; a worker runs it in the background and stores the result. Answers 202
    with the run waiting or in progress; poll …/readiness/progress until running is gone and latest is
    the new run. Fails with 409 readiness_running while a run waits or runs.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ReadinessPoll]
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

) -> Problem | ReadinessPoll | None:
    """ Run Send Readiness on a campaign

     Queues a run of every check; a worker runs it in the background and stores the result. Answers 202
    with the run waiting or in progress; poll …/readiness/progress until running is gone and latest is
    the new run. Fails with 409 readiness_running while a run waits or runs.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ReadinessPoll
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

) -> Response[Problem | ReadinessPoll]:
    """ Run Send Readiness on a campaign

     Queues a run of every check; a worker runs it in the background and stores the result. Answers 202
    with the run waiting or in progress; poll …/readiness/progress until running is gone and latest is
    the new run. Fails with 409 readiness_running while a run waits or runs.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ReadinessPoll]
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

) -> Problem | ReadinessPoll | None:
    """ Run Send Readiness on a campaign

     Queues a run of every check; a worker runs it in the background and stores the result. Answers 202
    with the run waiting or in progress; poll …/readiness/progress until running is gone and latest is
    the new run. Fails with 409 readiness_running while a run waits or runs.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ReadinessPoll
     """


    return (await asyncio_detailed(
        workspace=workspace,
campaign_id=campaign_id,
client=client,

    )).parsed
