from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.role_member_ref_status import RoleMemberRefStatus






T = TypeVar("T", bound="RoleMemberRef")



@_attrs_define
class RoleMemberRef:
    """ 
        Attributes:
            avatar_color (str):
            email (str):
            id (str): Membership id
            name (str):
            status (RoleMemberRefStatus):
     """

    avatar_color: str
    email: str
    id: str
    name: str
    status: RoleMemberRefStatus





    def to_dict(self) -> dict[str, Any]:
        avatar_color = self.avatar_color

        email = self.email

        id = self.id

        name = self.name

        status = self.status.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "avatarColor": avatar_color,
            "email": email,
            "id": id,
            "name": name,
            "status": status,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        avatar_color = d.pop("avatarColor")

        email = d.pop("email")

        id = d.pop("id")

        name = d.pop("name")

        status = RoleMemberRefStatus(d.pop("status"))




        role_member_ref = cls(
            avatar_color=avatar_color,
            email=email,
            id=id,
            name=name,
            status=status,
        )

        return role_member_ref

