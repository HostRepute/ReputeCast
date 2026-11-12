from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.scoring_tier_row_tier import ScoringTierRowTier






T = TypeVar("T", bound="ScoringTierRow")



@_attrs_define
class ScoringTierRow:
    """ 
        Attributes:
            cold (int):
            contacts (int):
            hot (int):
            tier (ScoringTierRowTier):
            warm (int):
     """

    cold: int
    contacts: int
    hot: int
    tier: ScoringTierRowTier
    warm: int





    def to_dict(self) -> dict[str, Any]:
        cold = self.cold

        contacts = self.contacts

        hot = self.hot

        tier = self.tier.value

        warm = self.warm


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "cold": cold,
            "contacts": contacts,
            "hot": hot,
            "tier": tier,
            "warm": warm,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        cold = d.pop("cold")

        contacts = d.pop("contacts")

        hot = d.pop("hot")

        tier = ScoringTierRowTier(d.pop("tier"))




        warm = d.pop("warm")

        scoring_tier_row = cls(
            cold=cold,
            contacts=contacts,
            hot=hot,
            tier=tier,
            warm=warm,
        )

        return scoring_tier_row

