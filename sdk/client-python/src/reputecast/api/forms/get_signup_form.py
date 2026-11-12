from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.signup_form import SignupForm
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    form: UUID,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/forms/{form}".format(workspace=quote(str(workspace), safe=""),form=quote(str(form), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | SignupForm:
    if response.status_code == 200:
        response_200 = SignupForm.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | SignupForm]:
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

) -> Response[Problem | SignupForm]:
    """ Get a signup form with its numbers for the last 30 days

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (UUID): Form id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SignupForm]
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

) -> Problem | SignupForm | None:
    """ Get a signup form with its numbers for the last 30 days

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (UUID): Form id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SignupForm
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

) -> Response[Problem | SignupForm]:
    """ Get a signup form with its numbers for the last 30 days

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (UUID): Form id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SignupForm]
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

) -> Problem | SignupForm | None:
    """ Get a signup form with its numbers for the last 30 days

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (UUID): Form id

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SignupForm
     """


    return (await asyncio_detailed(
        workspace=workspace,
form=form,
client=client,

    )).parsed
