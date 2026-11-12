from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.connect_setup_list import ConnectSetupList
from ...models.list_provider_setups_status import ListProviderSetupsStatus
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    adapter: str | Unset = UNSET,
    status: ListProviderSetupsStatus | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["adapter"] = adapter

    json_status: str | Unset = UNSET
    if not isinstance(status, Unset):
        json_status = status.value

    params["status"] = json_status


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/connect/setups".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> ConnectSetupList | Problem:
    if response.status_code == 200:
        response_200 = ConnectSetupList.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[ConnectSetupList | Problem]:
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
    adapter: str | Unset = UNSET,
    status: ListProviderSetupsStatus | Unset = UNSET,

) -> Response[ConnectSetupList | Problem]:
    """ The workspace's latest provider setups, newest first

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        adapter (str | Unset): Only setups of this adapter (ses, smtp…)
        status (ListProviderSetupsStatus | Unset): Only setups in this state

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ConnectSetupList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
adapter=adapter,
status=status,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    adapter: str | Unset = UNSET,
    status: ListProviderSetupsStatus | Unset = UNSET,

) -> ConnectSetupList | Problem | None:
    """ The workspace's latest provider setups, newest first

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        adapter (str | Unset): Only setups of this adapter (ses, smtp…)
        status (ListProviderSetupsStatus | Unset): Only setups in this state

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ConnectSetupList | Problem
     """


    return sync_detailed(
        workspace=workspace,
client=client,
adapter=adapter,
status=status,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    adapter: str | Unset = UNSET,
    status: ListProviderSetupsStatus | Unset = UNSET,

) -> Response[ConnectSetupList | Problem]:
    """ The workspace's latest provider setups, newest first

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        adapter (str | Unset): Only setups of this adapter (ses, smtp…)
        status (ListProviderSetupsStatus | Unset): Only setups in this state

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ConnectSetupList | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
adapter=adapter,
status=status,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    adapter: str | Unset = UNSET,
    status: ListProviderSetupsStatus | Unset = UNSET,

) -> ConnectSetupList | Problem | None:
    """ The workspace's latest provider setups, newest first

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        adapter (str | Unset): Only setups of this adapter (ses, smtp…)
        status (ListProviderSetupsStatus | Unset): Only setups in this state

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ConnectSetupList | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
adapter=adapter,
status=status,

    )).parsed
