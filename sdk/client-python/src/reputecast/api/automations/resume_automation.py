from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.automation_detail import AutomationDetail
from ...models.problem import Problem
from ...models.resume_input import ResumeInput
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    id: str,
    *,
    body: ResumeInput | Unset = UNSET,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/automations/{id}/resume".format(workspace=quote(str(workspace), safe=""),id=quote(str(id), safe=""),),
    }

    
    if not isinstance(body, Unset):
        _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> AutomationDetail | Problem:
    if response.status_code == 200:
        response_200 = AutomationDetail.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[AutomationDetail | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    id: str,
    *,
    client: AuthenticatedClient,
    body: ResumeInput | Unset = UNSET,

) -> Response[AutomationDetail | Problem]:
    """ Resume a paused automation, saying what happens to contacts whose wait ended meanwhile

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (str): Automation id
        body (ResumeInput | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AutomationDetail | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    id: str,
    *,
    client: AuthenticatedClient,
    body: ResumeInput | Unset = UNSET,

) -> AutomationDetail | Problem | None:
    """ Resume a paused automation, saying what happens to contacts whose wait ended meanwhile

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (str): Automation id
        body (ResumeInput | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AutomationDetail | Problem
     """


    return sync_detailed(
        workspace=workspace,
id=id,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    id: str,
    *,
    client: AuthenticatedClient,
    body: ResumeInput | Unset = UNSET,

) -> Response[AutomationDetail | Problem]:
    """ Resume a paused automation, saying what happens to contacts whose wait ended meanwhile

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (str): Automation id
        body (ResumeInput | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AutomationDetail | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    id: str,
    *,
    client: AuthenticatedClient,
    body: ResumeInput | Unset = UNSET,

) -> AutomationDetail | Problem | None:
    """ Resume a paused automation, saying what happens to contacts whose wait ended meanwhile

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (str): Automation id
        body (ResumeInput | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AutomationDetail | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
id=id,
client=client,
body=body,

    )).parsed
