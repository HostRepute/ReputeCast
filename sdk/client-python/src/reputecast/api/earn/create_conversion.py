from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.conversion_input import ConversionInput
from ...models.conversion_result import ConversionResult
from ...models.problem import Problem
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    body: ConversionInput,
    idempotency_key: str | Unset = UNSET,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    if not isinstance(idempotency_key, Unset):
        headers["Idempotency-Key"] = idempotency_key



    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/conversions".format(workspace=quote(str(workspace), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> ConversionResult | Problem:
    if response.status_code == 202:
        response_202 = ConversionResult.from_dict(response.json())



        return response_202

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[ConversionResult | Problem]:
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
    body: ConversionInput,
    idempotency_key: str | Unset = UNSET,

) -> Response[ConversionResult | Problem]:
    """ Record a conversion: an upgrade or an order in your product, with its value

     Credits the conversion to the last campaign or automation email the reader clicked within the
    attribution window: by click token (rc_click), else by email address. Answers 202 with how it was
    matched. Errors: missing_identity, invalid_value, invalid_currency, occurred_at_out_of_range (422),
    idempotency_key_reused (409), rate_limited (429, with Retry-After). Conversions are logged in Earn ›
    Settings › Attribution, not in the audit log.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        idempotency_key (str | Unset): Sending the same key again answers with the first
            conversion instead of recording a second
        body (ConversionInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ConversionResult | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
body=body,
idempotency_key=idempotency_key,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: ConversionInput,
    idempotency_key: str | Unset = UNSET,

) -> ConversionResult | Problem | None:
    """ Record a conversion: an upgrade or an order in your product, with its value

     Credits the conversion to the last campaign or automation email the reader clicked within the
    attribution window: by click token (rc_click), else by email address. Answers 202 with how it was
    matched. Errors: missing_identity, invalid_value, invalid_currency, occurred_at_out_of_range (422),
    idempotency_key_reused (409), rate_limited (429, with Retry-After). Conversions are logged in Earn ›
    Settings › Attribution, not in the audit log.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        idempotency_key (str | Unset): Sending the same key again answers with the first
            conversion instead of recording a second
        body (ConversionInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ConversionResult | Problem
     """


    return sync_detailed(
        workspace=workspace,
client=client,
body=body,
idempotency_key=idempotency_key,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: ConversionInput,
    idempotency_key: str | Unset = UNSET,

) -> Response[ConversionResult | Problem]:
    """ Record a conversion: an upgrade or an order in your product, with its value

     Credits the conversion to the last campaign or automation email the reader clicked within the
    attribution window: by click token (rc_click), else by email address. Answers 202 with how it was
    matched. Errors: missing_identity, invalid_value, invalid_currency, occurred_at_out_of_range (422),
    idempotency_key_reused (409), rate_limited (429, with Retry-After). Conversions are logged in Earn ›
    Settings › Attribution, not in the audit log.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        idempotency_key (str | Unset): Sending the same key again answers with the first
            conversion instead of recording a second
        body (ConversionInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ConversionResult | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
body=body,
idempotency_key=idempotency_key,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    body: ConversionInput,
    idempotency_key: str | Unset = UNSET,

) -> ConversionResult | Problem | None:
    """ Record a conversion: an upgrade or an order in your product, with its value

     Credits the conversion to the last campaign or automation email the reader clicked within the
    attribution window: by click token (rc_click), else by email address. Answers 202 with how it was
    matched. Errors: missing_identity, invalid_value, invalid_currency, occurred_at_out_of_range (422),
    idempotency_key_reused (409), rate_limited (429, with Retry-After). Conversions are logged in Earn ›
    Settings › Attribution, not in the audit log.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        idempotency_key (str | Unset): Sending the same key again answers with the first
            conversion instead of recording a second
        body (ConversionInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ConversionResult | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
body=body,
idempotency_key=idempotency_key,

    )).parsed
