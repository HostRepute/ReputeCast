from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="GiftMonthsCount")



@_attrs_define
class GiftMonthsCount:
    """ 
        Attributes:
            count (int):
            months (int):
     """

    count: int
    months: int





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        months = self.months


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "months": months,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        months = d.pop("months")

        gift_months_count = cls(
            count=count,
            months=months,
        )

        return gift_months_count

