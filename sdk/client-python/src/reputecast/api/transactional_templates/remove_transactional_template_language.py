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



def _get_kwargs(
    workspace: str,
    template: str,
    language: str,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "delete",
        "url": "/api/v1/workspaces/{workspace}/transactional/templates/{template}/languages/{language}".format(workspace=quote(str(workspace), safe=""),template=quote(str(template), safe=""),language=quote(str(language), safe=""),),
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
    template: str,
    language: str,
    *,
    client: AuthenticatedClient,

) -> Response[EmailLanguages | Problem]:
    """ Remove a language version from a template's draft

     Published versions keep theirs until the next publish.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        template (str): The template's ID, such as password-reset, or its UUID Example: password-
            reset.
        language (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[EmailLanguages | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
template=template,
language=language,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    template: str,
    language: str,
    *,
    client: AuthenticatedClient,

) -> EmailLanguages | Problem | None:
    """ Remove a language version from a template's draft

     Published versions keep theirs until the next publish.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        template (str): The template's ID, such as password-reset, or its UUID Example: password-
            reset.
        language (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        EmailLanguages | Problem
     """


    return sync_detailed(
        workspace=workspace,
template=template,
language=language,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    template: str,
    language: str,
    *,
    client: AuthenticatedClient,

) -> Response[EmailLanguages | Problem]:
    """ Remove a language version from a template's draft

     Published versions keep theirs until the next publish.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        template (str): The template's ID, such as password-reset, or its UUID Example: password-
            reset.
        language (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[EmailLanguages | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
template=template,
language=language,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    template: str,
    language: str,
    *,
    client: AuthenticatedClient,

) -> EmailLanguages | Problem | None:
    """ Remove a language version from a template's draft

     Published versions keep theirs until the next publish.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        template (str): The template's ID, such as password-reset, or its UUID Example: password-
            reset.
        language (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        EmailLanguages | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
template=template,
language=language,
client=client,

    )).parsed
