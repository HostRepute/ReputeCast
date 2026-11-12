from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.verification_form_input import VerificationFormInput
from ...models.verification_form_setting import VerificationFormSetting
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    form: str,
    *,
    body: VerificationFormInput,
    name: str | Unset = UNSET,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    params: dict[str, Any] = {}

    params["name"] = name


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/v1/workspaces/{workspace}/verification/forms/{form}".format(workspace=quote(str(workspace), safe=""),form=quote(str(form), safe=""),),
        "params": params,
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | VerificationFormSetting:
    if response.status_code == 200:
        response_200 = VerificationFormSetting.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | VerificationFormSetting]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    form: str,
    *,
    client: AuthenticatedClient,
    body: VerificationFormInput,
    name: str | Unset = UNSET,

) -> Response[Problem | VerificationFormSetting]:
    """ Turn a form's real-time address check on or off (also needs forms:edit; audited)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (str): A form's id, or subscribe_page
        name (str | Unset): The form's name, for the audit log
        body (VerificationFormInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | VerificationFormSetting]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
form=form,
body=body,
name=name,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    form: str,
    *,
    client: AuthenticatedClient,
    body: VerificationFormInput,
    name: str | Unset = UNSET,

) -> Problem | VerificationFormSetting | None:
    """ Turn a form's real-time address check on or off (also needs forms:edit; audited)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (str): A form's id, or subscribe_page
        name (str | Unset): The form's name, for the audit log
        body (VerificationFormInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | VerificationFormSetting
     """


    return sync_detailed(
        workspace=workspace,
form=form,
client=client,
body=body,
name=name,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    form: str,
    *,
    client: AuthenticatedClient,
    body: VerificationFormInput,
    name: str | Unset = UNSET,

) -> Response[Problem | VerificationFormSetting]:
    """ Turn a form's real-time address check on or off (also needs forms:edit; audited)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (str): A form's id, or subscribe_page
        name (str | Unset): The form's name, for the audit log
        body (VerificationFormInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | VerificationFormSetting]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
form=form,
body=body,
name=name,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    form: str,
    *,
    client: AuthenticatedClient,
    body: VerificationFormInput,
    name: str | Unset = UNSET,

) -> Problem | VerificationFormSetting | None:
    """ Turn a form's real-time address check on or off (also needs forms:edit; audited)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        form (str): A form's id, or subscribe_page
        name (str | Unset): The form's name, for the audit log
        body (VerificationFormInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | VerificationFormSetting
     """


    return (await asyncio_detailed(
        workspace=workspace,
form=form,
client=client,
body=body,
name=name,

    )).parsed
