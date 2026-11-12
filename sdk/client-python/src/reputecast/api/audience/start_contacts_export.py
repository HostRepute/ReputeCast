from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.audience_job import AudienceJob
from ...models.contact_export_input import ContactExportInput
from ...models.problem import Problem
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    body: ContactExportInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/contacts/jobs/export".format(workspace=quote(str(workspace), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> AudienceJob | Problem:
    if response.status_code == 202:
        response_202 = AudienceJob.from_dict(response.json())



        return response_202

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[AudienceJob | Problem]:
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
    body: ContactExportInput,

) -> Response[AudienceJob | Problem]:
    """ Export contacts in the background (logged)

     For any number of contacts: the file is written in the background (the jobs tray follows it) and
    kept 7 days for whoever started it. Needs subscribers:export; starting, finishing and each download
    are audited.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (ContactExportInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AudienceJob | Problem]
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
    body: ContactExportInput,

) -> AudienceJob | Problem | None:
    """ Export contacts in the background (logged)

     For any number of contacts: the file is written in the background (the jobs tray follows it) and
    kept 7 days for whoever started it. Needs subscribers:export; starting, finishing and each download
    are audited.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (ContactExportInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AudienceJob | Problem
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
    body: ContactExportInput,

) -> Response[AudienceJob | Problem]:
    """ Export contacts in the background (logged)

     For any number of contacts: the file is written in the background (the jobs tray follows it) and
    kept 7 days for whoever started it. Needs subscribers:export; starting, finishing and each download
    are audited.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (ContactExportInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[AudienceJob | Problem]
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
    body: ContactExportInput,

) -> AudienceJob | Problem | None:
    """ Export contacts in the background (logged)

     For any number of contacts: the file is written in the background (the jobs tray follows it) and
    kept 7 days for whoever started it. Needs subscribers:export; starting, finishing and each download
    are audited.

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        body (ContactExportInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        AudienceJob | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
body=body,

    )).parsed
