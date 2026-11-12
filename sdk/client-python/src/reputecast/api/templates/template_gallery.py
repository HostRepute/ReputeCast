from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.email_template_gallery import EmailTemplateGallery
from ...models.problem import Problem
from ...models.template_gallery_mode import TemplateGalleryMode
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    mode: TemplateGalleryMode | Unset = TemplateGalleryMode.REGULAR,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    json_mode: str | Unset = UNSET
    if not isinstance(mode, Unset):
        json_mode = mode.value

    params["mode"] = json_mode


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/templates".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> EmailTemplateGallery | Problem:
    if response.status_code == 200:
        response_200 = EmailTemplateGallery.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[EmailTemplateGallery | Problem]:
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
    mode: TemplateGalleryMode | Unset = TemplateGalleryMode.REGULAR,

) -> Response[EmailTemplateGallery | Problem]:
    """ Starter templates, saved templates, saved sections and the last sent campaign

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        mode (TemplateGalleryMode | Unset):  Default: TemplateGalleryMode.REGULAR.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[EmailTemplateGallery | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
mode=mode,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    mode: TemplateGalleryMode | Unset = TemplateGalleryMode.REGULAR,

) -> EmailTemplateGallery | Problem | None:
    """ Starter templates, saved templates, saved sections and the last sent campaign

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        mode (TemplateGalleryMode | Unset):  Default: TemplateGalleryMode.REGULAR.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        EmailTemplateGallery | Problem
     """


    return sync_detailed(
        workspace=workspace,
client=client,
mode=mode,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    mode: TemplateGalleryMode | Unset = TemplateGalleryMode.REGULAR,

) -> Response[EmailTemplateGallery | Problem]:
    """ Starter templates, saved templates, saved sections and the last sent campaign

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        mode (TemplateGalleryMode | Unset):  Default: TemplateGalleryMode.REGULAR.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[EmailTemplateGallery | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
mode=mode,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    mode: TemplateGalleryMode | Unset = TemplateGalleryMode.REGULAR,

) -> EmailTemplateGallery | Problem | None:
    """ Starter templates, saved templates, saved sections and the last sent campaign

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        mode (TemplateGalleryMode | Unset):  Default: TemplateGalleryMode.REGULAR.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        EmailTemplateGallery | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
mode=mode,

    )).parsed
