from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.gift_option import GiftOption
from ...models.gift_option_input import GiftOptionInput
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    id: UUID,
    *,
    body: GiftOptionInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/v1/workspaces/{workspace}/earn/gifts/options/{id}".format(workspace=quote(str(workspace), safe=""),id=quote(str(id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> GiftOption | Problem:
    if response.status_code == 200:
        response_200 = GiftOption.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[GiftOption | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    id: UUID,
    *,
    client: AuthenticatedClient,
    body: GiftOptionInput,

) -> Response[GiftOption | Problem]:
    """ Change a gift option; gifts already bought keep what they were sold as

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID): The gift option's id
        body (GiftOptionInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[GiftOption | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    id: UUID,
    *,
    client: AuthenticatedClient,
    body: GiftOptionInput,

) -> GiftOption | Problem | None:
    """ Change a gift option; gifts already bought keep what they were sold as

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID): The gift option's id
        body (GiftOptionInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        GiftOption | Problem
     """


    return sync_detailed(
        workspace=workspace,
id=id,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    id: UUID,
    *,
    client: AuthenticatedClient,
    body: GiftOptionInput,

) -> Response[GiftOption | Problem]:
    """ Change a gift option; gifts already bought keep what they were sold as

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID): The gift option's id
        body (GiftOptionInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[GiftOption | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    id: UUID,
    *,
    client: AuthenticatedClient,
    body: GiftOptionInput,

) -> GiftOption | Problem | None:
    """ Change a gift option; gifts already bought keep what they were sold as

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID): The gift option's id
        body (GiftOptionInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        GiftOption | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
id=id,
client=client,
body=body,

    )).parsed
