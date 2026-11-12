from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.automation_email_content import AutomationEmailContent
from ...models.automation_email_update_body import AutomationEmailUpdateBody
from ...models.problem import Problem
from typing import cast



def _get_kwargs(
    workspace: str,
    id: str,
    *,
    body: AutomationEmailUpdateBody,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/v1/workspaces/{workspace}/automation-emails/{id}".format(workspace=quote(str(workspace), safe=""),id=quote(str(id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> AutomationEmailContent | Problem:
    if response.status_code == 200:
        response_200 = AutomationEmailContent.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[AutomationEmailContent | Problem]:
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
    body: AutomationEmailUpdateBody,

) -> Response[AutomationEmailContent | Problem]:
    """ Save new content as the email's next version

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (str):
        body (AutomationEmailUpdateBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AutomationEmailContent | Problem]
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
    body: AutomationEmailUpdateBody,

) -> AutomationEmailContent | Problem | None:
    """ Save new content as the email's next version

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (str):
        body (AutomationEmailUpdateBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AutomationEmailContent | Problem
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
    body: AutomationEmailUpdateBody,

) -> Response[AutomationEmailContent | Problem]:
    """ Save new content as the email's next version

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (str):
        body (AutomationEmailUpdateBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AutomationEmailContent | Problem]
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
    body: AutomationEmailUpdateBody,

) -> AutomationEmailContent | Problem | None:
    """ Save new content as the email's next version

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (str):
        body (AutomationEmailUpdateBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AutomationEmailContent | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
id=id,
client=client,
body=body,

    )).parsed
