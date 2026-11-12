from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.deliverability_check_run import DeliverabilityCheckRun
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    run_id: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/deliverability/checks/{run_id}".format(workspace=quote(str(workspace), safe=""),run_id=quote(str(run_id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> DeliverabilityCheckRun | Problem:
    if response.status_code == 200:
        response_200 = DeliverabilityCheckRun.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[DeliverabilityCheckRun | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    run_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[DeliverabilityCheckRun | Problem]:
    """ Progress of a check run

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        run_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[DeliverabilityCheckRun | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
run_id=run_id,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    run_id: UUID,
    *,
    client: AuthenticatedClient,

) -> DeliverabilityCheckRun | Problem | None:
    """ Progress of a check run

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        run_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        DeliverabilityCheckRun | Problem
     """


    return sync_detailed(
        workspace=workspace,
run_id=run_id,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    run_id: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[DeliverabilityCheckRun | Problem]:
    """ Progress of a check run

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        run_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[DeliverabilityCheckRun | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
run_id=run_id,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    run_id: UUID,
    *,
    client: AuthenticatedClient,

) -> DeliverabilityCheckRun | Problem | None:
    """ Progress of a check run

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        run_id (UUID):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        DeliverabilityCheckRun | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
run_id=run_id,
client=client,

    )).parsed
