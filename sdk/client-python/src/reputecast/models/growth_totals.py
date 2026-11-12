from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="GrowthTotals")



@_attrs_define
class GrowthTotals:
    """ 
        Attributes:
            joined (int):
            left (int):
            net (int):
     """

    joined: int
    left: int
    net: int





    def to_dict(self) -> dict[str, Any]:
        joined = self.joined

        left = self.left

        net = self.net


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "joined": joined,
            "left": left,
            "net": net,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        joined = d.pop("joined")

        left = d.pop("left")

        net = d.pop("net")

        growth_totals = cls(
            joined=joined,
            left=left,
            net=net,
        )

        return growth_totals

