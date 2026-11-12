from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="RouteCampaignCondition")



@_attrs_define
class RouteCampaignCondition:
    """ 
        Attributes:
            count (int):
            field (str):
            values (list[str]):
     """

    count: int
    field: str
    values: list[str]





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        field = self.field

        values = self.values




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "field": field,
            "values": values,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        field = d.pop("field")

        values = cast(list[str], d.pop("values"))


        route_campaign_condition = cls(
            count=count,
            field=field,
            values=values,
        )

        return route_campaign_condition

