from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="TeamRoleRef")



@_attrs_define
class TeamRoleRef:
    """ 
        Attributes:
            builtin (bool):
            id (str):
            key (str):
            name (str):
     """

    builtin: bool
    id: str
    key: str
    name: str





    def to_dict(self) -> dict[str, Any]:
        builtin = self.builtin

        id = self.id

        key = self.key

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "builtin": builtin,
            "id": id,
            "key": key,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        builtin = d.pop("builtin")

        id = d.pop("id")

        key = d.pop("key")

        name = d.pop("name")

        team_role_ref = cls(
            builtin=builtin,
            id=id,
            key=key,
            name=name,
        )

        return team_role_ref

