from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="GrowthRest")



@_attrs_define
class GrowthRest:
    """ 
        Attributes:
            count (int):
            joins (int):
            human_open_rate (float | Unset):
     """

    count: int
    joins: int
    human_open_rate: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        joins = self.joins

        human_open_rate = self.human_open_rate


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "joins": joins,
        })
        if human_open_rate is not UNSET:
            field_dict["humanOpenRate"] = human_open_rate

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        joins = d.pop("joins")

        human_open_rate = d.pop("humanOpenRate", UNSET)

        growth_rest = cls(
            count=count,
            joins=joins,
            human_open_rate=human_open_rate,
        )

        return growth_rest

