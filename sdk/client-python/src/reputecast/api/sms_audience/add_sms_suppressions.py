from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.sms_suppression_add import SmsSuppressionAdd
from ...models.sms_suppressions_added import SmsSuppressionsAdded
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    body: SmsSuppressionAdd,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/sms/suppressions".format(workspace=quote(str(workspace), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | SmsSuppressionsAdded:
    if response.status_code == 201:
        response_201 = SmsSuppressionsAdded.from_dict(response.json())



        return response_201

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | SmsSuppressionsAdded]:
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
    body: SmsSuppressionAdd,

) -> Response[Problem | SmsSuppressionsAdded]:
    """ Put numbers on the SMS suppression list (logged)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (SmsSuppressionAdd):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SmsSuppressionsAdded]
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
    body: SmsSuppressionAdd,

) -> Problem | SmsSuppressionsAdded | None:
    """ Put numbers on the SMS suppression list (logged)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (SmsSuppressionAdd):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SmsSuppressionsAdded
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
    body: SmsSuppressionAdd,

) -> Response[Problem | SmsSuppressionsAdded]:
    """ Put numbers on the SMS suppression list (logged)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (SmsSuppressionAdd):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SmsSuppressionsAdded]
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
    body: SmsSuppressionAdd,

) -> Problem | SmsSuppressionsAdded | None:
    """ Put numbers on the SMS suppression list (logged)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (SmsSuppressionAdd):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SmsSuppressionsAdded
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
body=body,

    )).parsed
