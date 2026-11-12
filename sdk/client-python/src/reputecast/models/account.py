from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="Account")



@_attrs_define
class Account:
    """ 
        Attributes:
            hint (str):
            id (str):
            live (bool):
            name (str):
     """

    hint: str
    id: str
    live: bool
    name: str





    def to_dict(self) -> dict[str, Any]:
        hint = self.hint

        id = self.id

        live = self.live

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "hint": hint,
            "id": id,
            "live": live,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        hint = d.pop("hint")

        id = d.pop("id")

        live = d.pop("live")

        name = d.pop("name")

        account = cls(
            hint=hint,
            id=id,
            live=live,
            name=name,
        )

        return account

