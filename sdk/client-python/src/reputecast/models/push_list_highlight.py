from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PushListHighlight")



@_attrs_define
class PushListHighlight:
    """ 
        Attributes:
            count (int):
            list_id (str):
            name (str):
     """

    count: int
    list_id: str
    name: str





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        list_id = self.list_id

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "listId": list_id,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        list_id = d.pop("listId")

        name = d.pop("name")

        push_list_highlight = cls(
            count=count,
            list_id=list_id,
            name=name,
        )

        return push_list_highlight

