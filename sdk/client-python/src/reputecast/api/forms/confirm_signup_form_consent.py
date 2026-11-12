from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.signup_form_copy import SignupFormCopy
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    form: UUID,
    language: str,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/forms/{form}/copy/{language}/consent-reviewed".format(workspace=quote(str(workspace), safe=""),form=quote(str(form), safe=""),language=quote(str(language), safe=""),),
    }


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

) -> Response[Problem | SignupFormCopy]:
    """ Mark a language's consent text still right

     Clears "Review the translation" after the form's own consent text changed, keeping the translation
    and its version.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (UUID): Form id
        language (str): A workspace language (es)

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

) -> Problem | SignupFormCopy | None:
    """ Mark a language's consent text still right

     Clears "Review the translation" after the form's own consent text changed, keeping the translation
    and its version.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (UUID): Form id
        language (str): A workspace language (es)

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

    ).parsed

async def asyncio_detailed(
    workspace: str,
    form: UUID,
    language: str,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | SignupFormCopy]:
    """ Mark a language's consent text still right

     Clears "Review the translation" after the form's own consent text changed, keeping the translation
    and its version.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (UUID): Form id
        language (str): A workspace language (es)

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

) -> Problem | SignupFormCopy | None:
    """ Mark a language's consent text still right

     Clears "Review the translation" after the form's own consent text changed, keeping the translation
    and its version.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (UUID): Form id
        language (str): A workspace language (es)

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

    )).parsed
