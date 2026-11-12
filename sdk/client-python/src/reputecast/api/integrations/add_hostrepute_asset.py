from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.host_repute_add_asset_input_body import HostReputeAddAssetInputBody
from ...models.host_repute_connection import HostReputeConnection
from ...models.problem import Problem
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    body: HostReputeAddAssetInputBody,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/integrations/hostrepute/assets".format(workspace=quote(str(workspace), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> HostReputeConnection | Problem:
    if response.status_code == 200:
        response_200 = HostReputeConnection.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[HostReputeConnection | Problem]:
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
    body: HostReputeAddAssetInputBody,

) -> Response[HostReputeConnection | Problem]:
    """ Add a domain, IP or hostname to HostRepute monitoring

     Answers 409 with code monitor_limit_reached (meta: limit, used, team, asset, swap) when the
    HostRepute plan is full.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (HostReputeAddAssetInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[HostReputeConnection | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: HostReputeAddAssetInputBody,

) -> HostReputeConnection | Problem | None:
    """ Add a domain, IP or hostname to HostRepute monitoring

     Answers 409 with code monitor_limit_reached (meta: limit, used, team, asset, swap) when the
    HostRepute plan is full.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (HostReputeAddAssetInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        HostReputeConnection | Problem
     """


    return sync_detailed(
        workspace=workspace,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: HostReputeAddAssetInputBody,

) -> Response[HostReputeConnection | Problem]:
    """ Add a domain, IP or hostname to HostRepute monitoring

     Answers 409 with code monitor_limit_reached (meta: limit, used, team, asset, swap) when the
    HostRepute plan is full.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (HostReputeAddAssetInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[HostReputeConnection | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: HostReputeAddAssetInputBody,

) -> HostReputeConnection | Problem | None:
    """ Add a domain, IP or hostname to HostRepute monitoring

     Answers 409 with code monitor_limit_reached (meta: limit, used, team, asset, swap) when the
    HostRepute plan is full.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (HostReputeAddAssetInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        HostReputeConnection | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
body=body,

    )).parsed
