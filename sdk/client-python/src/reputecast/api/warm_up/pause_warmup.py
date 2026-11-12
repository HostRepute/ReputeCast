from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.warmup_plan_view import WarmupPlanView
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    plan_id: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/warmup/{plan_id}/pause".format(workspace=quote(str(workspace), safe=""),plan_id=quote(str(plan_id), safe=""),),
    }


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

) -> Response[Problem | WarmupPlanView]:
    """ Pause a warm-up plan

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        plan_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WarmupPlanView]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
plan_id=plan_id,

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

) -> Problem | WarmupPlanView | None:
    """ Pause a warm-up plan

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        plan_id (UUID):

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

    ).parsed

async def asyncio_detailed(
    workspace: str,
    plan_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | WarmupPlanView]:
    """ Pause a warm-up plan

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        plan_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WarmupPlanView]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
plan_id=plan_id,

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

) -> Problem | WarmupPlanView | None:
    """ Pause a warm-up plan

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        plan_id (UUID):

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

    )).parsed
