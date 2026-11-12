from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="VisibilityHiddenBlock")



@_attrs_define
class VisibilityHiddenBlock:
    """ 
        Attributes:
            id (str):
            name (str):
            why (str):
     """

    id: str
    name: str
    why: str





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        name = self.name

        why = self.why


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "name": name,
            "why": why,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        name = d.pop("name")

        why = d.pop("why")

        visibility_hidden_block = cls(
            id=id,
            name=name,
            why=why,
        )

        return visibility_hidden_block

