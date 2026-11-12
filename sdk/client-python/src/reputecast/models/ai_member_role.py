from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="AIMemberRole")



@_attrs_define
class AIMemberRole:
    """ 
        Attributes:
            name (str):
            role (str):
     """

    name: str
    role: str





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        role = self.role


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
            "role": role,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        role = d.pop("role")

        ai_member_role = cls(
            name=name,
            role=role,
        )

        return ai_member_role

