from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="RevenueMove")



@_attrs_define
class RevenueMove:
    """ 
        Attributes:
            cents (int):
            count (int):
     """

    cents: int
    count: int





    def to_dict(self) -> dict[str, Any]:
        cents = self.cents

        count = self.count


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "cents": cents,
            "count": count,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        cents = d.pop("cents")

        count = d.pop("count")

        revenue_move = cls(
            cents=cents,
            count=count,
        )

        return revenue_move

