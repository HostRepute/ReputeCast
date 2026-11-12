from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.integration_call_list import IntegrationCallList
from ...models.list_integration_calls_window import ListIntegrationCallsWindow
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    id: UUID,
    *,
    errors: bool | Unset = UNSET,
    window: ListIntegrationCallsWindow | Unset = ListIntegrationCallsWindow.VALUE_1,
    offset: int | Unset = 0,
    limit: int | Unset = 20,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["errors"] = errors

    json_window: str | Unset = UNSET
    if not isinstance(window, Unset):
        json_window = window.value

    params["window"] = json_window

    params["offset"] = offset

    params["limit"] = limit


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/integrations/connections/{id}/calls".format(workspace=quote(str(workspace), safe=""),id=quote(str(id), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> IntegrationCallList | Problem:
    if response.status_code == 200:
        response_200 = IntegrationCallList.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[IntegrationCallList | Problem]:
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
    errors: bool | Unset = UNSET,
    window: ListIntegrationCallsWindow | Unset = ListIntegrationCallsWindow.VALUE_1,
    offset: int | Unset = 0,
    limit: int | Unset = 20,

) -> Response[IntegrationCallList | Problem]:
    """ The 7-day call log: workflow, action, target, result and duration

     Addresses show to members with subscribers:view; others see "a contact".

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        errors (bool | Unset): Only errors
        window (ListIntegrationCallsWindow | Unset):  Default: ListIntegrationCallsWindow.VALUE_1.
        offset (int | Unset):  Default: 0.
        limit (int | Unset):  Default: 20.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[IntegrationCallList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
errors=errors,
window=window,
offset=offset,
limit=limit,

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
    errors: bool | Unset = UNSET,
    window: ListIntegrationCallsWindow | Unset = ListIntegrationCallsWindow.VALUE_1,
    offset: int | Unset = 0,
    limit: int | Unset = 20,

) -> IntegrationCallList | Problem | None:
    """ The 7-day call log: workflow, action, target, result and duration

     Addresses show to members with subscribers:view; others see "a contact".

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        errors (bool | Unset): Only errors
        window (ListIntegrationCallsWindow | Unset):  Default: ListIntegrationCallsWindow.VALUE_1.
        offset (int | Unset):  Default: 0.
        limit (int | Unset):  Default: 20.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        IntegrationCallList | Problem
     """


    return sync_detailed(
        workspace=workspace,
id=id,
client=client,
errors=errors,
window=window,
offset=offset,
limit=limit,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    id: UUID,
    *,
    client: AuthenticatedClient,
    errors: bool | Unset = UNSET,
    window: ListIntegrationCallsWindow | Unset = ListIntegrationCallsWindow.VALUE_1,
    offset: int | Unset = 0,
    limit: int | Unset = 20,

) -> Response[IntegrationCallList | Problem]:
    """ The 7-day call log: workflow, action, target, result and duration

     Addresses show to members with subscribers:view; others see "a contact".

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        errors (bool | Unset): Only errors
        window (ListIntegrationCallsWindow | Unset):  Default: ListIntegrationCallsWindow.VALUE_1.
        offset (int | Unset):  Default: 0.
        limit (int | Unset):  Default: 20.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[IntegrationCallList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
errors=errors,
window=window,
offset=offset,
limit=limit,

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
    errors: bool | Unset = UNSET,
    window: ListIntegrationCallsWindow | Unset = ListIntegrationCallsWindow.VALUE_1,
    offset: int | Unset = 0,
    limit: int | Unset = 20,

) -> IntegrationCallList | Problem | None:
    """ The 7-day call log: workflow, action, target, result and duration

     Addresses show to members with subscribers:view; others see "a contact".

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        errors (bool | Unset): Only errors
        window (ListIntegrationCallsWindow | Unset):  Default: ListIntegrationCallsWindow.VALUE_1.
        offset (int | Unset):  Default: 0.
        limit (int | Unset):  Default: 20.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        IntegrationCallList | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
id=id,
client=client,
errors=errors,
window=window,
offset=offset,
limit=limit,

    )).parsed
