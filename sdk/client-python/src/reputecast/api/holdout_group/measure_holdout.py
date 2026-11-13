from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.holdout_results import HoldoutResults
from ...models.problem import Problem
from typing import cast



def _get_kwargs(
    workspace: str,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/holdout/measure".format(workspace=quote(str(workspace), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> HoldoutResults | Problem:
    if response.status_code == 202:
        response_202 = HoldoutResults.from_dict(response.json())



        return response_202

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[HoldoutResults | Problem]:
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

) -> Response[HoldoutResults | Problem]:
    """ Recompute the holdout group's results

     At most once every 10 minutes (429 with Retry-After otherwise). The results say computing until the
    new ones are ready.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[HoldoutResults | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,

) -> HoldoutResults | Problem | None:
    """ Recompute the holdout group's results

     At most once every 10 minutes (429 with Retry-After otherwise). The results say computing until the
    new ones are ready.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        HoldoutResults | Problem
     """


    return sync_detailed(
        workspace=workspace,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,

) -> Response[HoldoutResults | Problem]:
    """ Recompute the holdout group's results

     At most once every 10 minutes (429 with Retry-After otherwise). The results say computing until the
    new ones are ready.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[HoldoutResults | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,

) -> HoldoutResults | Problem | None:
    """ Recompute the holdout group's results

     At most once every 10 minutes (429 with Retry-After otherwise). The results say computing until the
    new ones are ready.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        HoldoutResults | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,

    )).parsed
