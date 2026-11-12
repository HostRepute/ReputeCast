from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.deliverability_delist_input_body import DeliverabilityDelistInputBody
from ...models.deliverability_delisting_request import DeliverabilityDelistingRequest
from ...models.problem import Problem
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    body: DeliverabilityDelistInputBody,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/deliverability/delistings".format(workspace=quote(str(workspace), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> DeliverabilityDelistingRequest | Problem:
    if response.status_code == 200:
        response_200 = DeliverabilityDelistingRequest.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[DeliverabilityDelistingRequest | Problem]:
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
    body: DeliverabilityDelistInputBody,

) -> Response[DeliverabilityDelistingRequest | Problem]:
    """ Request delisting through HostRepute

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (DeliverabilityDelistInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[DeliverabilityDelistingRequest | Problem]
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
    body: DeliverabilityDelistInputBody,

) -> DeliverabilityDelistingRequest | Problem | None:
    """ Request delisting through HostRepute

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (DeliverabilityDelistInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        DeliverabilityDelistingRequest | Problem
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
    body: DeliverabilityDelistInputBody,

) -> Response[DeliverabilityDelistingRequest | Problem]:
    """ Request delisting through HostRepute

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (DeliverabilityDelistInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[DeliverabilityDelistingRequest | Problem]
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
    body: DeliverabilityDelistInputBody,

) -> DeliverabilityDelistingRequest | Problem | None:
    """ Request delisting through HostRepute

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (DeliverabilityDelistInputBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        DeliverabilityDelistingRequest | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
body=body,

    )).parsed
