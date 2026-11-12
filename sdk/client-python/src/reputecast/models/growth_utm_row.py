from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="GrowthUTMRow")



@_attrs_define
class GrowthUTMRow:
    """ 
        Attributes:
            campaign (str):
            joins (int):
            medium (str):
            source (str):
            human_open_rate (float | Unset):
     """

    campaign: str
    joins: int
    medium: str
    source: str
    human_open_rate: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        campaign = self.campaign

        joins = self.joins

        medium = self.medium

        source = self.source

        human_open_rate = self.human_open_rate


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "campaign": campaign,
            "joins": joins,
            "medium": medium,
            "source": source,
        })
        if human_open_rate is not UNSET:
            field_dict["humanOpenRate"] = human_open_rate

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        campaign = d.pop("campaign")

        joins = d.pop("joins")

        medium = d.pop("medium")

        source = d.pop("source")

        human_open_rate = d.pop("humanOpenRate", UNSET)

        growth_utm_row = cls(
            campaign=campaign,
            joins=joins,
            medium=medium,
            source=source,
            human_open_rate=human_open_rate,
        )

        return growth_utm_row

