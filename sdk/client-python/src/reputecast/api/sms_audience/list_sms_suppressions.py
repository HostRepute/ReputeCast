from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.list_sms_suppressions_reason import ListSmsSuppressionsReason
from ...models.problem import Problem
from ...models.sms_suppression_list import SmsSuppressionList
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    reason: ListSmsSuppressionsReason | Unset = UNSET,
    provider: str | Unset = UNSET,
    q: str | Unset = UNSET,
    page: int | Unset = 1,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    json_reason: str | Unset = UNSET
    if not isinstance(reason, Unset):
        json_reason = reason.value

    params["reason"] = json_reason

    params["provider"] = provider

    params["q"] = q

    params["page"] = page


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/sms/suppressions".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | SmsSuppressionList:
    if response.status_code == 200:
        response_200 = SmsSuppressionList.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | SmsSuppressionList]:
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
    reason: ListSmsSuppressionsReason | Unset = UNSET,
    provider: str | Unset = UNSET,
    q: str | Unset = UNSET,
    page: int | Unset = 1,

) -> Response[Problem | SmsSuppressionList]:
    """ The SMS suppression list, numbers partly masked

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        reason (ListSmsSuppressionsReason | Unset):
        provider (str | Unset): An SMS provider id
        q (str | Unset): Digits of the number
        page (int | Unset):  Default: 1.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SmsSuppressionList]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
reason=reason,
provider=provider,
q=q,
page=page,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    reason: ListSmsSuppressionsReason | Unset = UNSET,
    provider: str | Unset = UNSET,
    q: str | Unset = UNSET,
    page: int | Unset = 1,

) -> Problem | SmsSuppressionList | None:
    """ The SMS suppression list, numbers partly masked

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        reason (ListSmsSuppressionsReason | Unset):
        provider (str | Unset): An SMS provider id
        q (str | Unset): Digits of the number
        page (int | Unset):  Default: 1.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SmsSuppressionList
     """


    return sync_detailed(
        workspace=workspace,
client=client,
reason=reason,
provider=provider,
q=q,
page=page,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    reason: ListSmsSuppressionsReason | Unset = UNSET,
    provider: str | Unset = UNSET,
    q: str | Unset = UNSET,
    page: int | Unset = 1,

) -> Response[Problem | SmsSuppressionList]:
    """ The SMS suppression list, numbers partly masked

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        reason (ListSmsSuppressionsReason | Unset):
        provider (str | Unset): An SMS provider id
        q (str | Unset): Digits of the number
        page (int | Unset):  Default: 1.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | SmsSuppressionList]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
reason=reason,
provider=provider,
q=q,
page=page,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    reason: ListSmsSuppressionsReason | Unset = UNSET,
    provider: str | Unset = UNSET,
    q: str | Unset = UNSET,
    page: int | Unset = 1,

) -> Problem | SmsSuppressionList | None:
    """ The SMS suppression list, numbers partly masked

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        reason (ListSmsSuppressionsReason | Unset):
        provider (str | Unset): An SMS provider id
        q (str | Unset): Digits of the number
        page (int | Unset):  Default: 1.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | SmsSuppressionList
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
reason=reason,
provider=provider,
q=q,
page=page,

    )).parsed
