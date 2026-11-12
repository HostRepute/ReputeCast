from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.automation_event_body import AutomationEventBody
from ...models.automation_event_result import AutomationEventResult
from ...models.problem import Problem
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    body: AutomationEventBody,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/automations/events".format(workspace=quote(str(workspace), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> AutomationEventResult | Problem:
    if response.status_code == 202:
        response_202 = AutomationEventResult.from_dict(response.json())



        return response_202

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[AutomationEventResult | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: AutomationEventBody,

) -> Response[AutomationEventResult | Problem]:
    """ Send an API event for a contact (triggers, goals and branches listen to it)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (AutomationEventBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AutomationEventResult | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: AutomationEventBody,

) -> AutomationEventResult | Problem | None:
    """ Send an API event for a contact (triggers, goals and branches listen to it)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (AutomationEventBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AutomationEventResult | Problem
     """


    return sync_detailed(
        workspace=workspace,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: AutomationEventBody,

) -> Response[AutomationEventResult | Problem]:
    """ Send an API event for a contact (triggers, goals and branches listen to it)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (AutomationEventBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AutomationEventResult | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: AutomationEventBody,

) -> AutomationEventResult | Problem | None:
    """ Send an API event for a contact (triggers, goals and branches listen to it)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (AutomationEventBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AutomationEventResult | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
body=body,

    )).parsed
