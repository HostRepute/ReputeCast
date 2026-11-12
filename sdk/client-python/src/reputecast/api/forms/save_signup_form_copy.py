from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.signup_form_copy import SignupFormCopy
from ...models.signup_form_copy_text import SignupFormCopyText
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    form: UUID,
    language: str,
    *,
    body: SignupFormCopyText,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/v1/workspaces/{workspace}/forms/{form}/copy/{language}".format(workspace=quote(str(workspace), safe=""),form=quote(str(form), safe=""),language=quote(str(language), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | SignupFormCopy:
    if response.status_code == 200:
        response_200 = SignupFormCopy.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | SignupFormCopy]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    form: UUID,
    language: str,
    *,
    client: AuthenticatedClient,
    body: SignupFormCopyText,

) -> Response[Problem | SignupFormCopy]:
    """ Save a signup form's text in a language

     In the form's own language this is its content and consent text. Changing a consent text starts a
    new version of it in that language; consent evidence keeps the version and language shown.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (UUID): Form id
        language (str): A workspace language (es)
        body (SignupFormCopyText):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SignupFormCopy]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
form=form,
language=language,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    form: UUID,
    language: str,
    *,
    client: AuthenticatedClient,
    body: SignupFormCopyText,

) -> Problem | SignupFormCopy | None:
    """ Save a signup form's text in a language

     In the form's own language this is its content and consent text. Changing a consent text starts a
    new version of it in that language; consent evidence keeps the version and language shown.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (UUID): Form id
        language (str): A workspace language (es)
        body (SignupFormCopyText):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SignupFormCopy
     """


    return sync_detailed(
        workspace=workspace,
form=form,
language=language,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    form: UUID,
    language: str,
    *,
    client: AuthenticatedClient,
    body: SignupFormCopyText,

) -> Response[Problem | SignupFormCopy]:
    """ Save a signup form's text in a language

     In the form's own language this is its content and consent text. Changing a consent text starts a
    new version of it in that language; consent evidence keeps the version and language shown.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (UUID): Form id
        language (str): A workspace language (es)
        body (SignupFormCopyText):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SignupFormCopy]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
form=form,
language=language,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    form: UUID,
    language: str,
    *,
    client: AuthenticatedClient,
    body: SignupFormCopyText,

) -> Problem | SignupFormCopy | None:
    """ Save a signup form's text in a language

     In the form's own language this is its content and consent text. Changing a consent text starts a
    new version of it in that language; consent evidence keeps the version and language shown.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (UUID): Form id
        language (str): A workspace language (es)
        body (SignupFormCopyText):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SignupFormCopy
     """


    return (await asyncio_detailed(
        workspace=workspace,
form=form,
language=language,
client=client,
body=body,

    )).parsed
