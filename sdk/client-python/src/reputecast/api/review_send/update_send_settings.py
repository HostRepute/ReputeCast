from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.readiness_send_settings import ReadinessSendSettings
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    campaign_id: UUID,
    *,
    body: ReadinessSendSettings,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "patch",
        "url": "/api/v1/workspaces/{workspace}/campaigns/{campaign_id}/send-settings".format(workspace=quote(str(workspace), safe=""),campaign_id=quote(str(campaign_id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Any | Problem:
    if response.status_code == 204:
        response_204 = cast(Any, None)
        return response_204

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Any | Problem]:
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
    body: ReadinessSendSettings,

) -> Response[Any | Problem]:
    """ Change the auto-pause thresholds or the schedule

     The schedule is part of what an approval covers: changing it invalidates the approval.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        body (ReadinessSendSettings):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
campaign_id=campaign_id,
body=body,

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
    body: ReadinessSendSettings,

) -> Any | Problem | None:
    """ Change the auto-pause thresholds or the schedule

     The schedule is part of what an approval covers: changing it invalidates the approval.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        body (ReadinessSendSettings):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Problem
     """


    return sync_detailed(
        workspace=workspace,
campaign_id=campaign_id,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    campaign_id: UUID,
    *,
    client: AuthenticatedClient,
    body: ReadinessSendSettings,

) -> Response[Any | Problem]:
    """ Change the auto-pause thresholds or the schedule

     The schedule is part of what an approval covers: changing it invalidates the approval.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        body (ReadinessSendSettings):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
campaign_id=campaign_id,
body=body,

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
    body: ReadinessSendSettings,

) -> Any | Problem | None:
    """ Change the auto-pause thresholds or the schedule

     The schedule is part of what an approval covers: changing it invalidates the approval.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        body (ReadinessSendSettings):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
campaign_id=campaign_id,
client=client,
body=body,

    )).parsed
