from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="GrowthChartDay")



@_attrs_define
class GrowthChartDay:
    """ 
        Attributes:
            day (str):
            imports (int):
            joined (list[int]):
            left (list[int]):
     """

    day: str
    imports: int
    joined: list[int]
    left: list[int]





    def to_dict(self) -> dict[str, Any]:
        day = self.day

        imports = self.imports

        joined = self.joined



        left = self.left




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "day": day,
            "imports": imports,
            "joined": joined,
            "left": left,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        day = d.pop("day")

        imports = d.pop("imports")

        joined = cast(list[int], d.pop("joined"))


        left = cast(list[int], d.pop("left"))


        growth_chart_day = cls(
            day=day,
            imports=imports,
            joined=joined,
            left=left,
        )

        return growth_chart_day

