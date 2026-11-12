from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.remove_run_input import RemoveRunInput
from typing import cast



def _get_kwargs(
    workspace: str,
    id: str,
    run: str,
    *,
    body: RemoveRunInput,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}


    

    

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces/{workspace}/automations/{id}/runs/{run}/remove".format(workspace=quote(str(workspace), safe=""),id=quote(str(id), safe=""),run=quote(str(run), safe=""),),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Any | Problem:
    if response.status_code == 204:
        response_204 = cast(Any, None)
        return response_204

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Any | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    workspace: str,
    id: str,
    run: str,
    *,
    client: AuthenticatedClient,
    body: RemoveRunInput,

) -> Response[Any | Problem]:
    """ Take a contact out of a flow now (needs a reason, logged)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (str): Automation id
        run (str):
        body (RemoveRunInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
run=run,
body=body,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    id: str,
    run: str,
    *,
    client: AuthenticatedClient,
    body: RemoveRunInput,

) -> Any | Problem | None:
    """ Take a contact out of a flow now (needs a reason, logged)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (str): Automation id
        run (str):
        body (RemoveRunInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Problem
     """


    return sync_detailed(
        workspace=workspace,
id=id,
run=run,
client=client,
body=body,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    id: str,
    run: str,
    *,
    client: AuthenticatedClient,
    body: RemoveRunInput,

) -> Response[Any | Problem]:
    """ Take a contact out of a flow now (needs a reason, logged)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (str): Automation id
        run (str):
        body (RemoveRunInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Problem]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
id=id,
run=run,
body=body,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    id: str,
    run: str,
    *,
    client: AuthenticatedClient,
    body: RemoveRunInput,

) -> Any | Problem | None:
    """ Take a contact out of a flow now (needs a reason, logged)

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        id (str): Automation id
        run (str):
        body (RemoveRunInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Problem
     """


    return (await asyncio_detailed(
        workspace=workspace,
id=id,
run=run,
client=client,
body=body,

    )).parsed
