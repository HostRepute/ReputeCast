from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.sms_credentials_input import SmsCredentialsInput
from ...models.sms_overview import SmsOverview
from typing import cast
from uuid import UUID



def _get_kwargs(
    workspace: str,
    provider_id: UUID,
    *,
    body: SmsCredentialsInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/v1/workspaces/{workspace}/sms/providers/{provider_id}/credentials".format(workspace=quote(str(workspace), safe=""),provider_id=quote(str(provider_id), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | SmsOverview:
    if response.status_code == 200:
        response_200 = SmsOverview.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | SmsOverview]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    provider_id: UUID,
    *,
    client: AuthenticatedClient,
    body: SmsCredentialsInput,

) -> Response[Problem | SmsOverview]:
    """ Replace a provider's credentials (logged)

     Checked before they're stored, sealed, never returned. Held texts go once they work.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        provider_id (UUID): SMS provider id
        body (SmsCredentialsInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SmsOverview]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
provider_id=provider_id,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    provider_id: UUID,
    *,
    client: AuthenticatedClient,
    body: SmsCredentialsInput,

) -> Problem | SmsOverview | None:
    """ Replace a provider's credentials (logged)

     Checked before they're stored, sealed, never returned. Held texts go once they work.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        provider_id (UUID): SMS provider id
        body (SmsCredentialsInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SmsOverview
     """


    return sync_detailed(
        workspace=workspace,
provider_id=provider_id,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    provider_id: UUID,
    *,
    client: AuthenticatedClient,
    body: SmsCredentialsInput,

) -> Response[Problem | SmsOverview]:
    """ Replace a provider's credentials (logged)

     Checked before they're stored, sealed, never returned. Held texts go once they work.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        provider_id (UUID): SMS provider id
        body (SmsCredentialsInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SmsOverview]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
provider_id=provider_id,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    provider_id: UUID,
    *,
    client: AuthenticatedClient,
    body: SmsCredentialsInput,

) -> Problem | SmsOverview | None:
    """ Replace a provider's credentials (logged)

     Checked before they're stored, sealed, never returned. Held texts go once they work.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        provider_id (UUID): SMS provider id
        body (SmsCredentialsInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SmsOverview
     """


    return (await asyncio_detailed(
        workspace=workspace,
provider_id=provider_id,
client=client,
body=body,

    )).parsed
