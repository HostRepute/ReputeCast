from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ComposerMember")



@_attrs_define
class ComposerMember:
    """ 
        Attributes:
            color (str): Avatar colour
            id (str):
            name (str):
            role (str):
     """

    color: str
    id: str
    name: str
    role: str





    def to_dict(self) -> dict[str, Any]:
        color = self.color

        id = self.id

        name = self.name

        role = self.role


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "color": color,
            "id": id,
            "name": name,
            "role": role,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        color = d.pop("color")

        id = d.pop("id")

        name = d.pop("name")

        role = d.pop("role")

        composer_member = cls(
            color=color,
            id=id,
            name=name,
            role=role,
        )

        return composer_member

