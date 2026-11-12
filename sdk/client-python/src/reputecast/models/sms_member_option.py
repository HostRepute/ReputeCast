from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SmsMemberOption")



@_attrs_define
class SmsMemberOption:
    """ 
        Attributes:
            color (str):
            name (str):
            role (str):
            user_id (str):
     """

    color: str
    name: str
    role: str
    user_id: str





    def to_dict(self) -> dict[str, Any]:
        color = self.color

        name = self.name

        role = self.role

        user_id = self.user_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "color": color,
            "name": name,
            "role": role,
            "userId": user_id,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        color = d.pop("color")

        name = d.pop("name")

        role = d.pop("role")

        user_id = d.pop("userId")

        sms_member_option = cls(
            color=color,
            name=name,
            role=role,
            user_id=user_id,
        )

        return sms_member_option

