from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="MarketingCap")



@_attrs_define
class MarketingCap:
    """ 
        Attributes:
            count (int):
            days (int):
     """

    count: int
    days: int





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        days = self.days


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "days": days,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        days = d.pop("days")

        marketing_cap = cls(
            count=count,
            days=days,
        )

        return marketing_cap

