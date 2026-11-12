from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.health_tier_card_tier import HealthTierCardTier
from ..types import UNSET, Unset






T = TypeVar("T", bound="HealthTierCard")



@_attrs_define
class HealthTierCard:
    """ 
        Attributes:
            count (int):
            good (bool):
            pct (float): Share of all contacts, percent
            tier (HealthTierCardTier):
            delta_pts (float | Unset):
     """

    count: int
    good: bool
    pct: float
    tier: HealthTierCardTier
    delta_pts: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        good = self.good

        pct = self.pct

        tier = self.tier.value

        delta_pts = self.delta_pts


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "good": good,
            "pct": pct,
            "tier": tier,
        })
        if delta_pts is not UNSET:
            field_dict["deltaPts"] = delta_pts

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        good = d.pop("good")

        pct = d.pop("pct")

        tier = HealthTierCardTier(d.pop("tier"))




        delta_pts = d.pop("deltaPts", UNSET)

        health_tier_card = cls(
            count=count,
            good=good,
            pct=pct,
            tier=tier,
            delta_pts=delta_pts,
        )

        return health_tier_card

