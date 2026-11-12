from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.warmup_hold_decision import WarmupHoldDecision
from ...models.warmup_hold_decision_result import WarmupHoldDecisionResult
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    plan_id: UUID,
    campaign_id: UUID,
    *,
    body: WarmupHoldDecision,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/warmup/{plan_id}/held/{campaign_id}".format(workspace=quote(str(workspace), safe=""),plan_id=quote(str(plan_id), safe=""),campaign_id=quote(str(campaign_id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | WarmupHoldDecisionResult:
    if response.status_code == 200:
        response_200 = WarmupHoldDecisionResult.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | WarmupHoldDecisionResult]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    plan_id: UUID,
    campaign_id: UUID,
    *,
    client: AuthenticatedClient,
    body: WarmupHoldDecision,

) -> Response[Problem | WarmupHoldDecisionResult]:
    """ Wait for tomorrow's cap, move or cancel a share held at the warm-up cap

     A share held at a warming provider's daily cap waits for tomorrow's cap by itself. Move sends it
    through another provider once it passes the same checks as a failover target; cancel drops the held
    rest of this share only. Cancelling needs campaigns:pause; waiting and moving resume the send and
    need campaigns:send.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        plan_id (UUID):
        campaign_id (UUID):
        body (WarmupHoldDecision):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WarmupHoldDecisionResult]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
plan_id=plan_id,
campaign_id=campaign_id,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    plan_id: UUID,
    campaign_id: UUID,
    *,
    client: AuthenticatedClient,
    body: WarmupHoldDecision,

) -> Problem | WarmupHoldDecisionResult | None:
    """ Wait for tomorrow's cap, move or cancel a share held at the warm-up cap

     A share held at a warming provider's daily cap waits for tomorrow's cap by itself. Move sends it
    through another provider once it passes the same checks as a failover target; cancel drops the held
    rest of this share only. Cancelling needs campaigns:pause; waiting and moving resume the send and
    need campaigns:send.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        plan_id (UUID):
        campaign_id (UUID):
        body (WarmupHoldDecision):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WarmupHoldDecisionResult
     """


    return sync_detailed(
        workspace=workspace,
plan_id=plan_id,
campaign_id=campaign_id,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    plan_id: UUID,
    campaign_id: UUID,
    *,
    client: AuthenticatedClient,
    body: WarmupHoldDecision,

) -> Response[Problem | WarmupHoldDecisionResult]:
    """ Wait for tomorrow's cap, move or cancel a share held at the warm-up cap

     A share held at a warming provider's daily cap waits for tomorrow's cap by itself. Move sends it
    through another provider once it passes the same checks as a failover target; cancel drops the held
    rest of this share only. Cancelling needs campaigns:pause; waiting and moving resume the send and
    need campaigns:send.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        plan_id (UUID):
        campaign_id (UUID):
        body (WarmupHoldDecision):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WarmupHoldDecisionResult]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
plan_id=plan_id,
campaign_id=campaign_id,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    plan_id: UUID,
    campaign_id: UUID,
    *,
    client: AuthenticatedClient,
    body: WarmupHoldDecision,

) -> Problem | WarmupHoldDecisionResult | None:
    """ Wait for tomorrow's cap, move or cancel a share held at the warm-up cap

     A share held at a warming provider's daily cap waits for tomorrow's cap by itself. Move sends it
    through another provider once it passes the same checks as a failover target; cancel drops the held
    rest of this share only. Cancelling needs campaigns:pause; waiting and moving resume the send and
    need campaigns:send.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        plan_id (UUID):
        campaign_id (UUID):
        body (WarmupHoldDecision):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WarmupHoldDecisionResult
     """


    return (await asyncio_detailed(
        workspace=workspace,
plan_id=plan_id,
campaign_id=campaign_id,
client=client,
body=body,

    )).parsed
