from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="GrowthImportItem")



@_attrs_define
class GrowthImportItem:
    """ 
        Attributes:
            count (int):
            day (str):
            key (str):
            name (str):
     """

    count: int
    day: str
    key: str
    name: str





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        day = self.day

        key = self.key

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "day": day,
            "key": key,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        day = d.pop("day")

        key = d.pop("key")

        name = d.pop("name")

        growth_import_item = cls(
            count=count,
            day=day,
            key=key,
            name=name,
        )

        return growth_import_item

