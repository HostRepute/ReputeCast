from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.email_languages import EmailLanguages
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    campaign_id: UUID,
    language: str,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/campaigns/{campaign_id}/languages/{language}/mark-translated".format(workspace=quote(str(workspace), safe=""),campaign_id=quote(str(campaign_id), safe=""),language=quote(str(language), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> EmailLanguages | Problem:
    if response.status_code == 200:
        response_200 = EmailLanguages.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[EmailLanguages | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    campaign_id: UUID,
    language: str,
    *,
    client: AuthenticatedClient,

) -> Response[EmailLanguages | Problem]:
    """ Mark a Draft language version translated, so it is sent

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        language (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[EmailLanguages | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
campaign_id=campaign_id,
language=language,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    campaign_id: UUID,
    language: str,
    *,
    client: AuthenticatedClient,

) -> EmailLanguages | Problem | None:
    """ Mark a Draft language version translated, so it is sent

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        language (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        EmailLanguages | Problem
     """


    return sync_detailed(
        workspace=workspace,
campaign_id=campaign_id,
language=language,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    campaign_id: UUID,
    language: str,
    *,
    client: AuthenticatedClient,

) -> Response[EmailLanguages | Problem]:
    """ Mark a Draft language version translated, so it is sent

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        language (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[EmailLanguages | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
campaign_id=campaign_id,
language=language,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    campaign_id: UUID,
    language: str,
    *,
    client: AuthenticatedClient,

) -> EmailLanguages | Problem | None:
    """ Mark a Draft language version translated, so it is sent

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        campaign_id (UUID):
        language (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        EmailLanguages | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
campaign_id=campaign_id,
language=language,
client=client,

    )).parsed
