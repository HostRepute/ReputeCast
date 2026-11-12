from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="RouteTestSplit")



@_attrs_define
class RouteTestSplit:
    """ 
        Attributes:
            short (str):
            weight (int):
     """

    short: str
    weight: int





    def to_dict(self) -> dict[str, Any]:
        short = self.short

        weight = self.weight


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "short": short,
            "weight": weight,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        short = d.pop("short")

        weight = d.pop("weight")

        route_test_split = cls(
            short=short,
            weight=weight,
        )

        return route_test_split

