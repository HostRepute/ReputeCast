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

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/forms/{form}/copy".format(workspace=quote(str(workspace), safe=""),form=quote(str(form), safe=""),),
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
    *,
    client: AuthenticatedClient,

) -> Response[Problem | SignupFormCopy]:
    """ Get a signup form's copy in every language

     The form's language mode, each language's status and text, its consent text's version and the latest
    consent given in it.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (UUID): Form id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SignupFormCopy]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
form=form,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    form: UUID,
    *,
    client: AuthenticatedClient,

) -> Problem | SignupFormCopy | None:
    """ Get a signup form's copy in every language

     The form's language mode, each language's status and text, its consent text's version and the latest
    consent given in it.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (UUID): Form id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SignupFormCopy
     """


    return sync_detailed(
        workspace=workspace,
form=form,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    form: UUID,
    *,
    client: AuthenticatedClient,

) -> Response[Problem | SignupFormCopy]:
    """ Get a signup form's copy in every language

     The form's language mode, each language's status and text, its consent text's version and the latest
    consent given in it.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (UUID): Form id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SignupFormCopy]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
form=form,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    form: UUID,
    *,
    client: AuthenticatedClient,

) -> Problem | SignupFormCopy | None:
    """ Get a signup form's copy in every language

     The form's language mode, each language's status and text, its consent text's version and the latest
    consent given in it.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (UUID): Form id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SignupFormCopy
     """


    return (await asyncio_detailed(
        workspace=workspace,
form=form,
client=client,

    )).parsed
