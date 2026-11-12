from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.email_template_preview import EmailTemplatePreview
from ...models.problem import Problem
from typing import cast



def _get_kwargs(
    workspace: str,
    template_id: str,

) -> dict[str, Any]:
    

    

    

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/templates/{template_id}/preview".format(workspace=quote(str(workspace), safe=""),template_id=quote(str(template_id), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> EmailTemplatePreview | Problem:
    if response.status_code == 200:
        response_200 = EmailTemplatePreview.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[EmailTemplatePreview | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    template_id: str,
    *,
    client: AuthenticatedClient,

) -> Response[EmailTemplatePreview | Problem]:
    """ A template's email, for the gallery's preview of it

     Compiled on the server (a starter in the workspace's brand kit), personalized for a reader without a
    first name so the fallbacks show, with its links, scripts and form controls taken out: a picture of
    the email, not a page to use.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        template_id (str): A template id, or starter:<key> for a built-in starter

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[EmailTemplatePreview | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
template_id=template_id,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    template_id: str,
    *,
    client: AuthenticatedClient,

) -> EmailTemplatePreview | Problem | None:
    """ A template's email, for the gallery's preview of it

     Compiled on the server (a starter in the workspace's brand kit), personalized for a reader without a
    first name so the fallbacks show, with its links, scripts and form controls taken out: a picture of
    the email, not a page to use.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        template_id (str): A template id, or starter:<key> for a built-in starter

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        EmailTemplatePreview | Problem
     """


    return sync_detailed(
        workspace=workspace,
template_id=template_id,
client=client,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    template_id: str,
    *,
    client: AuthenticatedClient,

) -> Response[EmailTemplatePreview | Problem]:
    """ A template's email, for the gallery's preview of it

     Compiled on the server (a starter in the workspace's brand kit), personalized for a reader without a
    first name so the fallbacks show, with its links, scripts and form controls taken out: a picture of
    the email, not a page to use.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        template_id (str): A template id, or starter:<key> for a built-in starter

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[EmailTemplatePreview | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
template_id=template_id,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    template_id: str,
    *,
    client: AuthenticatedClient,

) -> EmailTemplatePreview | Problem | None:
    """ A template's email, for the gallery's preview of it

     Compiled on the server (a starter in the workspace's brand kit), personalized for a reader without a
    first name so the fallbacks show, with its links, scripts and form controls taken out: a picture of
    the email, not a page to use.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        template_id (str): A template id, or starter:<key> for a built-in starter

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        EmailTemplatePreview | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
template_id=template_id,
client=client,

    )).parsed
