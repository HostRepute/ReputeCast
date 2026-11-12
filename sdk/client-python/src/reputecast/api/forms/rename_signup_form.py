from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.signup_form import SignupForm
from ...models.signup_form_rename_input import SignupFormRenameInput
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    form: UUID,
    *,
    body: SignupFormRenameInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/forms/{form}/rename".format(workspace=quote(str(workspace), safe=""),form=quote(str(form), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
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
    body: SignupFormRenameInput,

) -> Response[Problem | SignupForm]:
    """ Rename a signup form

     The form's hosted page follows the new name (/subscribe/<name>); embed code keeps working.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (UUID): Form id
        body (SignupFormRenameInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SignupForm]
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
    body: SignupFormRenameInput,

) -> Problem | SignupForm | None:
    """ Rename a signup form

     The form's hosted page follows the new name (/subscribe/<name>); embed code keeps working.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (UUID): Form id
        body (SignupFormRenameInput):

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
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    form: UUID,
    *,
    client: AuthenticatedClient,
    body: SignupFormRenameInput,

) -> Response[Problem | SignupForm]:
    """ Rename a signup form

     The form's hosted page follows the new name (/subscribe/<name>); embed code keeps working.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (UUID): Form id
        body (SignupFormRenameInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SignupForm]
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
    body: SignupFormRenameInput,

) -> Problem | SignupForm | None:
    """ Rename a signup form

     The form's hosted page follows the new name (/subscribe/<name>); embed code keeps working.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (UUID): Form id
        body (SignupFormRenameInput):

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
body=body,

    )).parsed
