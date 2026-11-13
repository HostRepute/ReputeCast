from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.lead_ads_form_detail import LeadAdsFormDetail
from ...models.problem import Problem
from typing import cast



def _get_kwargs(
    workspace: str,
    form_id: str,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/lead-ads/forms/{form_id}".format(workspace=quote(str(workspace), safe=""),form_id=quote(str(form_id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> LeadAdsFormDetail | Problem:
    if response.status_code == 200:
        response_200 = LeadAdsFormDetail.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[LeadAdsFormDetail | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    form_id: str,
    *,
    client: AuthenticatedClient,

) -> Response[LeadAdsFormDetail | Problem]:
    """ A lead form's questions, consent checkboxes and mapping (read from Meta when it answers)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form_id (str): The lead form's id in ReputeCast

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[LeadAdsFormDetail | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
form_id=form_id,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    form_id: str,
    *,
    client: AuthenticatedClient,

) -> LeadAdsFormDetail | Problem | None:
    """ A lead form's questions, consent checkboxes and mapping (read from Meta when it answers)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form_id (str): The lead form's id in ReputeCast

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        LeadAdsFormDetail | Problem
     """


    return sync_detailed(
        workspace=workspace,
form_id=form_id,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    form_id: str,
    *,
    client: AuthenticatedClient,

) -> Response[LeadAdsFormDetail | Problem]:
    """ A lead form's questions, consent checkboxes and mapping (read from Meta when it answers)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form_id (str): The lead form's id in ReputeCast

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[LeadAdsFormDetail | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
form_id=form_id,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    form_id: str,
    *,
    client: AuthenticatedClient,

) -> LeadAdsFormDetail | Problem | None:
    """ A lead form's questions, consent checkboxes and mapping (read from Meta when it answers)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form_id (str): The lead form's id in ReputeCast

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        LeadAdsFormDetail | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
form_id=form_id,
client=client,

    )).parsed
