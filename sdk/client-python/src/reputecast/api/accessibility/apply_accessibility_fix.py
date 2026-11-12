from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.accessibility_fix_input import AccessibilityFixInput
from ...models.accessibility_fixed import AccessibilityFixed
from ...models.problem import Problem
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    body: AccessibilityFixInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/accessibility/fix".format(workspace=quote(str(workspace), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> AccessibilityFixed | Problem:
    if response.status_code == 200:
        response_200 = AccessibilityFixed.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[AccessibilityFixed | Problem]:
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
    body: AccessibilityFixInput,

) -> Response[AccessibilityFixed | Problem]:
    """ Apply an accessibility fix to MJML

     The composer's fixes: alt text, decorative images, link text, a color, heading levels, a text size,
    and a Text block's role. Returns the changed MJML; nothing is saved, the composer's autosave stores
    it.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (AccessibilityFixInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AccessibilityFixed | Problem]
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
    body: AccessibilityFixInput,

) -> AccessibilityFixed | Problem | None:
    """ Apply an accessibility fix to MJML

     The composer's fixes: alt text, decorative images, link text, a color, heading levels, a text size,
    and a Text block's role. Returns the changed MJML; nothing is saved, the composer's autosave stores
    it.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (AccessibilityFixInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AccessibilityFixed | Problem
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
    body: AccessibilityFixInput,

) -> Response[AccessibilityFixed | Problem]:
    """ Apply an accessibility fix to MJML

     The composer's fixes: alt text, decorative images, link text, a color, heading levels, a text size,
    and a Text block's role. Returns the changed MJML; nothing is saved, the composer's autosave stores
    it.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (AccessibilityFixInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AccessibilityFixed | Problem]
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
    body: AccessibilityFixInput,

) -> AccessibilityFixed | Problem | None:
    """ Apply an accessibility fix to MJML

     The composer's fixes: alt text, decorative images, link text, a color, heading levels, a text size,
    and a Text block's role. Returns the changed MJML; nothing is saved, the composer's autosave stores
    it.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (AccessibilityFixInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AccessibilityFixed | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
body=body,

    )).parsed
