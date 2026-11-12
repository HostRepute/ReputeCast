from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="CampaignUpgradeLine")



@_attrs_define
class CampaignUpgradeLine:
    """ 
        Attributes:
            count (int):
            period (str):
            tier (str):
            offer (str | Unset):
     """

    count: int
    period: str
    tier: str
    offer: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        period = self.period

        tier = self.tier

        offer = self.offer


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "period": period,
            "tier": tier,
        })
        if offer is not UNSET:
            field_dict["offer"] = offer

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        period = d.pop("period")

        tier = d.pop("tier")

        offer = d.pop("offer", UNSET)

        campaign_upgrade_line = cls(
            count=count,
            period=period,
            tier=tier,
            offer=offer,
        )

        return campaign_upgrade_line

