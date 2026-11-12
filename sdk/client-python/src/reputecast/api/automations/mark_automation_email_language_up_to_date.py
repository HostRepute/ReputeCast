from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.automation_language_mark import AutomationLanguageMark
from ...models.automation_language_saved import AutomationLanguageSaved
from ...models.problem import Problem
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    id: UUID,
    language: str,
    *,
    body: AutomationLanguageMark,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/automation-emails/{id}/languages/{language}/mark-up-to-date".format(workspace=quote(str(workspace), safe=""),id=quote(str(id), safe=""),language=quote(str(language), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> AutomationLanguageSaved | Problem:
    if response.status_code == 200:
        response_200 = AutomationLanguageSaved.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[AutomationLanguageSaved | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    id: UUID,
    language: str,
    *,
    client: AuthenticatedClient,
    body: AutomationLanguageMark,

) -> Response[AutomationLanguageSaved | Problem]:
    """ Mark an automation email's Outdated language version up to date

     Writing makes the email's next version, with every language version carried over; a step sends it
    once the flow is published.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        language (str):
        body (AutomationLanguageMark):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AutomationLanguageSaved | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
language=language,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    id: UUID,
    language: str,
    *,
    client: AuthenticatedClient,
    body: AutomationLanguageMark,

) -> AutomationLanguageSaved | Problem | None:
    """ Mark an automation email's Outdated language version up to date

     Writing makes the email's next version, with every language version carried over; a step sends it
    once the flow is published.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        language (str):
        body (AutomationLanguageMark):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AutomationLanguageSaved | Problem
     """


    return sync_detailed(
        workspace=workspace,
id=id,
language=language,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    id: UUID,
    language: str,
    *,
    client: AuthenticatedClient,
    body: AutomationLanguageMark,

) -> Response[AutomationLanguageSaved | Problem]:
    """ Mark an automation email's Outdated language version up to date

     Writing makes the email's next version, with every language version carried over; a step sends it
    once the flow is published.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        language (str):
        body (AutomationLanguageMark):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AutomationLanguageSaved | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
language=language,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    id: UUID,
    language: str,
    *,
    client: AuthenticatedClient,
    body: AutomationLanguageMark,

) -> AutomationLanguageSaved | Problem | None:
    """ Mark an automation email's Outdated language version up to date

     Writing makes the email's next version, with every language version carried over; a step sends it
    once the flow is published.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (UUID):
        language (str):
        body (AutomationLanguageMark):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AutomationLanguageSaved | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
id=id,
language=language,
client=client,
body=body,

    )).parsed
