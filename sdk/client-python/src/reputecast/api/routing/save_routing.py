from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.routing_config import RoutingConfig
from ...models.routing_save_input import RoutingSaveInput
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    body: RoutingSaveInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/v1/workspaces/{workspace}/routing".format(workspace=quote(str(workspace), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | RoutingConfig:
    if response.status_code == 200:
        response_200 = RoutingConfig.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | RoutingConfig]:
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
    body: RoutingSaveInput,

) -> Response[Problem | RoutingConfig]:
    """ Replace the routing of one or more streams

     Validates splits, fallbacks and conditions against the connected providers. A Broadcast route that a
    provider's terms don't allow is refused with code bulk_terms unless acknowledgeBulkTerms is set; the
    decision is audited.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (RoutingSaveInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | RoutingConfig]
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
    body: RoutingSaveInput,

) -> Problem | RoutingConfig | None:
    """ Replace the routing of one or more streams

     Validates splits, fallbacks and conditions against the connected providers. A Broadcast route that a
    provider's terms don't allow is refused with code bulk_terms unless acknowledgeBulkTerms is set; the
    decision is audited.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (RoutingSaveInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | RoutingConfig
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
    body: RoutingSaveInput,

) -> Response[Problem | RoutingConfig]:
    """ Replace the routing of one or more streams

     Validates splits, fallbacks and conditions against the connected providers. A Broadcast route that a
    provider's terms don't allow is refused with code bulk_terms unless acknowledgeBulkTerms is set; the
    decision is audited.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (RoutingSaveInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | RoutingConfig]
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
    body: RoutingSaveInput,

) -> Problem | RoutingConfig | None:
    """ Replace the routing of one or more streams

     Validates splits, fallbacks and conditions against the connected providers. A Broadcast route that a
    provider's terms don't allow is refused with code bulk_terms unless acknowledgeBulkTerms is set; the
    decision is audited.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (RoutingSaveInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | RoutingConfig
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
body=body,

    )).parsed
