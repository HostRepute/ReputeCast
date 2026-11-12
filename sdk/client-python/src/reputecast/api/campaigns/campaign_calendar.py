from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.campaign_calendar import CampaignCalendar
from ...models.problem import Problem
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    from_: str,
    to: str,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["from"] = from_

    params["to"] = to


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/campaigns/calendar".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> CampaignCalendar | Problem:
    if response.status_code == 200:
        response_200 = CampaignCalendar.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[CampaignCalendar | Problem]:
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
    from_: str,
    to: str,

) -> Response[CampaignCalendar | Problem]:
    """ Scheduled and sent campaigns and digest issues between two days

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        from_ (str): First day (the workspace's time zone)
        to (str): Day after the last

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[CampaignCalendar | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
from_=from_,
to=to,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    from_: str,
    to: str,

) -> CampaignCalendar | Problem | None:
    """ Scheduled and sent campaigns and digest issues between two days

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        from_ (str): First day (the workspace's time zone)
        to (str): Day after the last

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        CampaignCalendar | Problem
     """


    return sync_detailed(
        workspace=workspace,
client=client,
from_=from_,
to=to,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    from_: str,
    to: str,

) -> Response[CampaignCalendar | Problem]:
    """ Scheduled and sent campaigns and digest issues between two days

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        from_ (str): First day (the workspace's time zone)
        to (str): Day after the last

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[CampaignCalendar | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
from_=from_,
to=to,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    from_: str,
    to: str,

) -> CampaignCalendar | Problem | None:
    """ Scheduled and sent campaigns and digest issues between two days

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        from_ (str): First day (the workspace's time zone)
        to (str): Day after the last

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        CampaignCalendar | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
from_=from_,
to=to,

    )).parsed
