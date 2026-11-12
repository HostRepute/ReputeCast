from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.warmup_plan_view import WarmupPlanView
from ...models.warmup_settings import WarmupSettings
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    plan_id: UUID,
    *,
    body: WarmupSettings,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/v1/workspaces/{workspace}/warmup/{plan_id}".format(workspace=quote(str(workspace), safe=""),plan_id=quote(str(plan_id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | WarmupPlanView:
    if response.status_code == 200:
        response_200 = WarmupPlanView.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | WarmupPlanView]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    plan_id: UUID,
    *,
    client: AuthenticatedClient,
    body: WarmupSettings,

) -> Response[Problem | WarmupPlanView]:
    """ Change a plan's length, ordering and guards

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        plan_id (UUID):
        body (WarmupSettings):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WarmupPlanView]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
plan_id=plan_id,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    plan_id: UUID,
    *,
    client: AuthenticatedClient,
    body: WarmupSettings,

) -> Problem | WarmupPlanView | None:
    """ Change a plan's length, ordering and guards

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        plan_id (UUID):
        body (WarmupSettings):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WarmupPlanView
     """


    return sync_detailed(
        workspace=workspace,
plan_id=plan_id,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    plan_id: UUID,
    *,
    client: AuthenticatedClient,
    body: WarmupSettings,

) -> Response[Problem | WarmupPlanView]:
    """ Change a plan's length, ordering and guards

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        plan_id (UUID):
        body (WarmupSettings):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WarmupPlanView]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
plan_id=plan_id,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    plan_id: UUID,
    *,
    client: AuthenticatedClient,
    body: WarmupSettings,

) -> Problem | WarmupPlanView | None:
    """ Change a plan's length, ordering and guards

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        plan_id (UUID):
        body (WarmupSettings):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WarmupPlanView
     """


    return (await asyncio_detailed(
        workspace=workspace,
plan_id=plan_id,
client=client,
body=body,

    )).parsed
