from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.lead_ads_activity import LeadAdsActivity
from ...models.problem import Problem
from typing import cast



def _get_kwargs(
    workspace: str,
    id: str,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/lead-ads/connections/{id}/activity".format(workspace=quote(str(workspace), safe=""),id=quote(str(id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> LeadAdsActivity | Problem:
    if response.status_code == 200:
        response_200 = LeadAdsActivity.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[LeadAdsActivity | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    id: str,
    *,
    client: AuthenticatedClient,

) -> Response[LeadAdsActivity | Problem]:
    """ The connection's last 50 leads: when, which form, what happened and the contact (no answers)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (str): The connection's id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[LeadAdsActivity | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    id: str,
    *,
    client: AuthenticatedClient,

) -> LeadAdsActivity | Problem | None:
    """ The connection's last 50 leads: when, which form, what happened and the contact (no answers)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (str): The connection's id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        LeadAdsActivity | Problem
     """


    return sync_detailed(
        workspace=workspace,
id=id,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    id: str,
    *,
    client: AuthenticatedClient,

) -> Response[LeadAdsActivity | Problem]:
    """ The connection's last 50 leads: when, which form, what happened and the contact (no answers)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (str): The connection's id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[LeadAdsActivity | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    id: str,
    *,
    client: AuthenticatedClient,

) -> LeadAdsActivity | Problem | None:
    """ The connection's last 50 leads: when, which form, what happened and the contact (no answers)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (str): The connection's id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        LeadAdsActivity | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
id=id,
client=client,

    )).parsed
