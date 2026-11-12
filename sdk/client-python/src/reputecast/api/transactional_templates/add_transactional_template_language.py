from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.add_language_input import AddLanguageInput
from ...models.email_languages import EmailLanguages
from ...models.problem import Problem
from typing import cast



def _get_kwargs(
    workspace: str,
    template: str,
    *,
    body: AddLanguageInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/transactional/templates/{template}/languages".format(workspace=quote(str(workspace), safe=""),template=quote(str(template), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> EmailLanguages | Problem:
    if response.status_code == 201:
        response_201 = EmailLanguages.from_dict(response.json())



        return response_201

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
    *,
    client: AuthenticatedClient,
    body: AddLanguageInput,

) -> Response[EmailLanguages | Problem]:
    """ Add a language version (a Draft) to a template's draft

     It starts from a copy of the source (layout and text) or from the layout only. A Draft is never
    sent: its recipients get the fallback until it is marked translated. Edit it with ?language= on the
    template and its draft.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        template (str): The template's ID, such as password-reset, or its UUID Example: password-
            reset.
        body (AddLanguageInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[EmailLanguages | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
template=template,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    template: str,
    *,
    client: AuthenticatedClient,
    body: AddLanguageInput,

) -> EmailLanguages | Problem | None:
    """ Add a language version (a Draft) to a template's draft

     It starts from a copy of the source (layout and text) or from the layout only. A Draft is never
    sent: its recipients get the fallback until it is marked translated. Edit it with ?language= on the
    template and its draft.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        template (str): The template's ID, such as password-reset, or its UUID Example: password-
            reset.
        body (AddLanguageInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        EmailLanguages | Problem
     """


    return sync_detailed(
        workspace=workspace,
template=template,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    template: str,
    *,
    client: AuthenticatedClient,
    body: AddLanguageInput,

) -> Response[EmailLanguages | Problem]:
    """ Add a language version (a Draft) to a template's draft

     It starts from a copy of the source (layout and text) or from the layout only. A Draft is never
    sent: its recipients get the fallback until it is marked translated. Edit it with ?language= on the
    template and its draft.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        template (str): The template's ID, such as password-reset, or its UUID Example: password-
            reset.
        body (AddLanguageInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[EmailLanguages | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
template=template,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    template: str,
    *,
    client: AuthenticatedClient,
    body: AddLanguageInput,

) -> EmailLanguages | Problem | None:
    """ Add a language version (a Draft) to a template's draft

     It starts from a copy of the source (layout and text) or from the layout only. A Draft is never
    sent: its recipients get the fallback until it is marked translated. Edit it with ?language= on the
    template and its draft.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        template (str): The template's ID, such as password-reset, or its UUID Example: password-
            reset.
        body (AddLanguageInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        EmailLanguages | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
template=template,
client=client,
body=body,

    )).parsed
