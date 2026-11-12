from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="EarnEventCount")



@_attrs_define
class EarnEventCount:
    """ 
        Attributes:
            count (int):
            type_ (str):
     """

    count: int
    type_: str





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        type_ = self.type_


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "type": type_,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        type_ = d.pop("type")

        earn_event_count = cls(
            count=count,
            type_=type_,
        )

        return earn_event_count

