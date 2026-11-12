from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.set_sidebar_section_section import SetSidebarSectionSection
from ...models.workspace_sidebar import WorkspaceSidebar
from ...models.workspace_sidebar_section_input import WorkspaceSidebarSectionInput
from typing import cast



def _get_kwargs(
    workspace: str,
    section: SetSidebarSectionSection,
    *,
    body: WorkspaceSidebarSectionInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/v1/workspaces/{workspace}/settings/sidebar/{section}".format(workspace=quote(str(workspace), safe=""),section=quote(str(section), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | WorkspaceSidebar:
    if response.status_code == 200:
        response_200 = WorkspaceSidebar.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | WorkspaceSidebar]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    section: SetSidebarSectionSection,
    *,
    client: AuthenticatedClient,
    body: WorkspaceSidebarSectionInput,

) -> Response[Problem | WorkspaceSidebar]:
    """ Show or hide a section in the sidebar (members can still open it by link)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        section (SetSidebarSectionSection): The section
        body (WorkspaceSidebarSectionInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WorkspaceSidebar]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
section=section,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    section: SetSidebarSectionSection,
    *,
    client: AuthenticatedClient,
    body: WorkspaceSidebarSectionInput,

) -> Problem | WorkspaceSidebar | None:
    """ Show or hide a section in the sidebar (members can still open it by link)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        section (SetSidebarSectionSection): The section
        body (WorkspaceSidebarSectionInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WorkspaceSidebar
     """


    return sync_detailed(
        workspace=workspace,
section=section,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    section: SetSidebarSectionSection,
    *,
    client: AuthenticatedClient,
    body: WorkspaceSidebarSectionInput,

) -> Response[Problem | WorkspaceSidebar]:
    """ Show or hide a section in the sidebar (members can still open it by link)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        section (SetSidebarSectionSection): The section
        body (WorkspaceSidebarSectionInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | WorkspaceSidebar]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
section=section,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    section: SetSidebarSectionSection,
    *,
    client: AuthenticatedClient,
    body: WorkspaceSidebarSectionInput,

) -> Problem | WorkspaceSidebar | None:
    """ Show or hide a section in the sidebar (members can still open it by link)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        section (SetSidebarSectionSection): The section
        body (WorkspaceSidebarSectionInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | WorkspaceSidebar
     """


    return (await asyncio_detailed(
        workspace=workspace,
section=section,
client=client,
body=body,

    )).parsed
