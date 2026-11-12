from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.list_members_status import ListMembersStatus
from ...models.list_members_twofa import ListMembersTwofa
from ...models.problem import Problem
from ...models.team_member_list import TeamMemberList
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    workspace: str,
    *,
    q: str | Unset = UNSET,
    role: str | Unset = UNSET,
    status: ListMembersStatus | Unset = UNSET,
    twofa: ListMembersTwofa | Unset = UNSET,

) -> dict[str, Any]:
    

    

    params: dict[str, Any] = {}

    params["q"] = q

    params["role"] = role

    json_status: str | Unset = UNSET
    if not isinstance(status, Unset):
        json_status = status.value

    params["status"] = json_status

    json_twofa: str | Unset = UNSET
    if not isinstance(twofa, Unset):
        json_twofa = twofa.value

    params["twofa"] = json_twofa


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/workspaces/{workspace}/members".format(workspace=quote(str(workspace), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Problem | TeamMemberList:
    if response.status_code == 200:
        response_200 = TeamMemberList.from_dict(response.json())



        return response_200

    response_default = Problem.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Problem | TeamMemberList]:
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
    q: str | Unset = UNSET,
    role: str | Unset = UNSET,
    status: ListMembersStatus | Unset = UNSET,
    twofa: ListMembersTwofa | Unset = UNSET,

) -> Response[Problem | TeamMemberList]:
    """ List members with role, scope, sign-in methods and two-factor status

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): Search name or email
        role (str | Unset): Role id or key
        status (ListMembersStatus | Unset):
        twofa (ListMembersTwofa | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TeamMemberList]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
q=q,
role=role,
status=status,
twofa=twofa,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    workspace: str,
    *,
    client: AuthenticatedClient,
    q: str | Unset = UNSET,
    role: str | Unset = UNSET,
    status: ListMembersStatus | Unset = UNSET,
    twofa: ListMembersTwofa | Unset = UNSET,

) -> Problem | TeamMemberList | None:
    """ List members with role, scope, sign-in methods and two-factor status

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): Search name or email
        role (str | Unset): Role id or key
        status (ListMembersStatus | Unset):
        twofa (ListMembersTwofa | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TeamMemberList
     """


    return sync_detailed(
        workspace=workspace,
client=client,
q=q,
role=role,
status=status,
twofa=twofa,

    ).parsed

async def asyncio_detailed(
    workspace: str,
    *,
    client: AuthenticatedClient,
    q: str | Unset = UNSET,
    role: str | Unset = UNSET,
    status: ListMembersStatus | Unset = UNSET,
    twofa: ListMembersTwofa | Unset = UNSET,

) -> Response[Problem | TeamMemberList]:
    """ List members with role, scope, sign-in methods and two-factor status

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): Search name or email
        role (str | Unset): Role id or key
        status (ListMembersStatus | Unset):
        twofa (ListMembersTwofa | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TeamMemberList]
     """


    kwargs = _get_kwargs(
        workspace=workspace,
q=q,
role=role,
status=status,
twofa=twofa,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    workspace: str,
    *,
    client: AuthenticatedClient,
    q: str | Unset = UNSET,
    role: str | Unset = UNSET,
    status: ListMembersStatus | Unset = UNSET,
    twofa: ListMembersTwofa | Unset = UNSET,

) -> Problem | TeamMemberList | None:
    """ List members with role, scope, sign-in methods and two-factor status

    Args:
        workspace (str): Workspace slug Example: acme-newsletter.
        q (str | Unset): Search name or email
        role (str | Unset): Role id or key
        status (ListMembersStatus | Unset):
        twofa (ListMembersTwofa | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TeamMemberList
     """


    return (await asyncio_detailed(
        workspace=workspace,
client=client,
q=q,
role=role,
status=status,
twofa=twofa,

    )).parsed
