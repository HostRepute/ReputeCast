from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="LiveCostByRoute")



@_attrs_define
class LiveCostByRoute:
    """ 
        Attributes:
            planned (float):
            total (float):
     """

    planned: float
    total: float





    def to_dict(self) -> dict[str, Any]:
        planned = self.planned

        total = self.total


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "planned": planned,
            "total": total,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        planned = d.pop("planned")

        total = d.pop("total")

        live_cost_by_route = cls(
            planned=planned,
            total=total,
        )

        return live_cost_by_route

