from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.signup_form_copy import SignupFormCopy
from ...models.signup_form_language_input import SignupFormLanguageInput
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    form: UUID,
    *,
    body: SignupFormLanguageInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/v1/workspaces/{workspace}/forms/{form}/language".format(workspace=quote(str(workspace), safe=""),form=quote(str(form), safe=""),),
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
    *,
    client: AuthenticatedClient,
    body: SignupFormLanguageInput,

) -> Response[Problem | SignupFormCopy]:
    """ Pick which language a signup form shows

     page: the lang of the page it's on; browser: the visitor's browser; fixed: always one language. A
    language whose copy isn't Live shows the fallback.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (UUID): Form id
        body (SignupFormLanguageInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SignupFormCopy]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
form=form,
body=body,

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
    body: SignupFormLanguageInput,

) -> Problem | SignupFormCopy | None:
    """ Pick which language a signup form shows

     page: the lang of the page it's on; browser: the visitor's browser; fixed: always one language. A
    language whose copy isn't Live shows the fallback.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (UUID): Form id
        body (SignupFormLanguageInput):

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
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    form: UUID,
    *,
    client: AuthenticatedClient,
    body: SignupFormLanguageInput,

) -> Response[Problem | SignupFormCopy]:
    """ Pick which language a signup form shows

     page: the lang of the page it's on; browser: the visitor's browser; fixed: always one language. A
    language whose copy isn't Live shows the fallback.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (UUID): Form id
        body (SignupFormLanguageInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SignupFormCopy]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
form=form,
body=body,

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
    body: SignupFormLanguageInput,

) -> Problem | SignupFormCopy | None:
    """ Pick which language a signup form shows

     page: the lang of the page it's on; browser: the visitor's browser; fixed: always one language. A
    language whose copy isn't Live shows the fallback.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (UUID): Form id
        body (SignupFormLanguageInput):

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
body=body,

    )).parsed
