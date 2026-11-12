from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.overview_tier_tier import OverviewTierTier






T = TypeVar("T", bound="OverviewTier")



@_attrs_define
class OverviewTier:
    """ 
        Attributes:
            count (int):
            description (str):
            pct (float): Share of all contacts, percent
            tier (OverviewTierTier):
     """

    count: int
    description: str
    pct: float
    tier: OverviewTierTier





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        description = self.description

        pct = self.pct

        tier = self.tier.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "description": description,
            "pct": pct,
            "tier": tier,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        description = d.pop("description")

        pct = d.pop("pct")

        tier = OverviewTierTier(d.pop("tier"))




        overview_tier = cls(
            count=count,
            description=description,
            pct=pct,
            tier=tier,
        )

        return overview_tier

