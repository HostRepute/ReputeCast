from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.push_send_input import PushSendInput
from ...models.push_send_result import PushSendResult
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    campaign_id: UUID,
    *,
    body: PushSendInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/push-campaigns/{campaign_id}/send".format(workspace=quote(str(workspace), safe=""),campaign_id=quote(str(campaign_id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | PushSendResult:
    if response.status_code == 200:
        response_200 = PushSendResult.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | PushSendResult]:
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
    body: PushSendInput,

) -> Response[Problem | PushSendResult]:
    """ Send or schedule a push campaign as reviewed (logged)

     Needs campaigns:send and push:send; the workspace's approval rules apply.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        body (PushSendInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | PushSendResult]
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
    body: PushSendInput,

) -> Problem | PushSendResult | None:
    """ Send or schedule a push campaign as reviewed (logged)

     Needs campaigns:send and push:send; the workspace's approval rules apply.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        body (PushSendInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | PushSendResult
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
    body: PushSendInput,

) -> Response[Problem | PushSendResult]:
    """ Send or schedule a push campaign as reviewed (logged)

     Needs campaigns:send and push:send; the workspace's approval rules apply.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        body (PushSendInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | PushSendResult]
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
    body: PushSendInput,

) -> Problem | PushSendResult | None:
    """ Send or schedule a push campaign as reviewed (logged)

     Needs campaigns:send and push:send; the workspace's approval rules apply.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        body (PushSendInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | PushSendResult
     """


    return (await asyncio_detailed(
        workspace=workspace,
campaign_id=campaign_id,
client=client,
body=body,

    )).parsed
