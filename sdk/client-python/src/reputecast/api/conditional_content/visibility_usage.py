from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.problem import Problem
from ...models.visibility_usage import VisibilityUsage
from ...models.visibility_usage_kind import VisibilityUsageKind
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    kind: VisibilityUsageKind | Unset = UNSET,
    ref: str | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    json_kind: str | Unset = UNSET
    if not isinstance(kind, Unset):
        json_kind = kind.value

    params["kind"] = json_kind

    params["ref"] = ref


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/visibility/usage".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | VisibilityUsage:
    if response.status_code == 200:
        response_200 = VisibilityUsage.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | VisibilityUsage]:
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
    kind: VisibilityUsageKind | Unset = UNSET,
    ref: str | Unset = UNSET,

) -> Response[Problem | VisibilityUsage]:
    """ Conditional blocks that use a segment, list, field or tag

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        kind (VisibilityUsageKind | Unset): What the rules refer to
        ref (str | Unset): A segment or list id, a field key or a tag

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | VisibilityUsage]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
kind=kind,
ref=ref,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    kind: VisibilityUsageKind | Unset = UNSET,
    ref: str | Unset = UNSET,

) -> Problem | VisibilityUsage | None:
    """ Conditional blocks that use a segment, list, field or tag

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        kind (VisibilityUsageKind | Unset): What the rules refer to
        ref (str | Unset): A segment or list id, a field key or a tag

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | VisibilityUsage
     """


    return sync_detailed(
        workspace=workspace,
client=client,
kind=kind,
ref=ref,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    kind: VisibilityUsageKind | Unset = UNSET,
    ref: str | Unset = UNSET,

) -> Response[Problem | VisibilityUsage]:
    """ Conditional blocks that use a segment, list, field or tag

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        kind (VisibilityUsageKind | Unset): What the rules refer to
        ref (str | Unset): A segment or list id, a field key or a tag

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | VisibilityUsage]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
kind=kind,
ref=ref,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    kind: VisibilityUsageKind | Unset = UNSET,
    ref: str | Unset = UNSET,

) -> Problem | VisibilityUsage | None:
    """ Conditional blocks that use a segment, list, field or tag

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        kind (VisibilityUsageKind | Unset): What the rules refer to
        ref (str | Unset): A segment or list id, a field key or a tag

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | VisibilityUsage
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
kind=kind,
ref=ref,

    )).parsed
