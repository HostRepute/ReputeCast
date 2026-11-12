from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.role_member_ref import RoleMemberRef





T = TypeVar("T", bound="RoleSummary")



@_attrs_define
class RoleSummary:
    """ 
        Attributes:
            active (int):
            builtin (bool):
            created_at (datetime.datetime):
            deactivated (int):
            description (str):
            id (str):
            invited (int):
            key (str):
            members (list[RoleMemberRef]):
            name (str):
            permissions (list[str]):
            sensitive (bool):
            updated_at (datetime.datetime):
            created_by (str | Unset):
     """

    active: int
    builtin: bool
    created_at: datetime.datetime
    deactivated: int
    description: str
    id: str
    invited: int
    key: str
    members: list[RoleMemberRef]
    name: str
    permissions: list[str]
    sensitive: bool
    updated_at: datetime.datetime
    created_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.role_member_ref import RoleMemberRef # noqa: PLC0415
        active = self.active

        builtin = self.builtin

        created_at = self.created_at.isoformat()

        deactivated = self.deactivated

        description = self.description

        id = self.id

        invited = self.invited

        key = self.key

        members = []
        for members_item_data in self.members:
            members_item = members_item_data.to_dict()
            members.append(members_item)



        name = self.name

        permissions = self.permissions



        sensitive = self.sensitive

        updated_at = self.updated_at.isoformat()

        created_by = self.created_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "active": active,
            "builtin": builtin,
            "createdAt": created_at,
            "deactivated": deactivated,
            "description": description,
            "id": id,
            "invited": invited,
            "key": key,
            "members": members,
            "name": name,
            "permissions": permissions,
            "sensitive": sensitive,
            "updatedAt": updated_at,
        })
        if created_by is not UNSET:
            field_dict["createdBy"] = created_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.role_member_ref import RoleMemberRef # noqa: PLC0415
        d = dict(src_dict)
        active = d.pop("active")

        builtin = d.pop("builtin")

        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        deactivated = d.pop("deactivated")

        description = d.pop("description")

        id = d.pop("id")

        invited = d.pop("invited")

        key = d.pop("key")

        members = []
        _members = d.pop("members")
        for members_item_data in (_members):
            members_item = RoleMemberRef.from_dict(members_item_data)



            members.append(members_item)


        name = d.pop("name")

        permissions = cast(list[str], d.pop("permissions"))


        sensitive = d.pop("sensitive")

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))




        created_by = d.pop("createdBy", UNSET)

        role_summary = cls(
            active=active,
            builtin=builtin,
            created_at=created_at,
            deactivated=deactivated,
            description=description,
            id=id,
            invited=invited,
            key=key,
            members=members,
            name=name,
            permissions=permissions,
            sensitive=sensitive,
            updated_at=updated_at,
            created_by=created_by,
        )

        return role_summary

