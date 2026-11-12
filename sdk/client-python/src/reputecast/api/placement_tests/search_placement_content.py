from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.placement_content_list import PlacementContentList
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    q: str | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["q"] = q


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/placement/content".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> PlacementContentList | Problem:
    if response.status_code == 200:
        response_200 = PlacementContentList.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[PlacementContentList | Problem]:
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
    q: str | Unset = UNSET,

) -> Response[PlacementContentList | Problem]:
    """ Campaigns, templates and transactional templates a test can send

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): Words of the name

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[PlacementContentList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
q=q,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    q: str | Unset = UNSET,

) -> PlacementContentList | Problem | None:
    """ Campaigns, templates and transactional templates a test can send

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): Words of the name

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        PlacementContentList | Problem
     """


    return sync_detailed(
        workspace=workspace,
client=client,
q=q,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    q: str | Unset = UNSET,

) -> Response[PlacementContentList | Problem]:
    """ Campaigns, templates and transactional templates a test can send

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): Words of the name

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[PlacementContentList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
q=q,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    q: str | Unset = UNSET,

) -> PlacementContentList | Problem | None:
    """ Campaigns, templates and transactional templates a test can send

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): Words of the name

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        PlacementContentList | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
q=q,

    )).parsed
