from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.team_member_api_key_kind import TeamMemberAPIKeyKind
from ..models.team_member_api_key_status import TeamMemberAPIKeyStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="TeamMemberAPIKey")



@_attrs_define
class TeamMemberAPIKey:
    """ 
        Attributes:
            id (str):
            kind (TeamMemberAPIKeyKind):
            masked (str):
            name (str):
            permissions (list[str]):
            scope (str):
            status (TeamMemberAPIKeyStatus):
            last_used_at (datetime.datetime | Unset):
     """

    id: str
    kind: TeamMemberAPIKeyKind
    masked: str
    name: str
    permissions: list[str]
    scope: str
    status: TeamMemberAPIKeyStatus
    last_used_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        kind = self.kind.value

        masked = self.masked

        name = self.name

        permissions = self.permissions



        scope = self.scope

        status = self.status.value

        last_used_at: str | Unset = UNSET
        if not isinstance(self.last_used_at, Unset):
            last_used_at = self.last_used_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "kind": kind,
            "masked": masked,
            "name": name,
            "permissions": permissions,
            "scope": scope,
            "status": status,
        })
        if last_used_at is not UNSET:
            field_dict["lastUsedAt"] = last_used_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        kind = TeamMemberAPIKeyKind(d.pop("kind"))




        masked = d.pop("masked")

        name = d.pop("name")

        permissions = cast(list[str], d.pop("permissions"))


        scope = d.pop("scope")

        status = TeamMemberAPIKeyStatus(d.pop("status"))




        _last_used_at = d.pop("lastUsedAt", UNSET)
        last_used_at: datetime.datetime | Unset
        if isinstance(_last_used_at,  Unset):
            last_used_at = UNSET
        else:
            last_used_at = datetime.datetime.fromisoformat(_last_used_at)




        team_member_api_key = cls(
            id=id,
            kind=kind,
            masked=masked,
            name=name,
            permissions=permissions,
            scope=scope,
            status=status,
            last_used_at=last_used_at,
        )

        return team_member_api_key

