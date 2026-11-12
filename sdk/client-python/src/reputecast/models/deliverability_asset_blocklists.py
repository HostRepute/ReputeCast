from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.deliverability_asset_blocklists_via import DeliverabilityAssetBlocklistsVia
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="DeliverabilityAssetBlocklists")



@_attrs_define
class DeliverabilityAssetBlocklists:
    """ 
        Attributes:
            high_impact (bool):
            listed (int):
            listings (list[str]):
            sources (int):
            via (DeliverabilityAssetBlocklistsVia):
            checked_at (datetime.datetime | Unset):
     """

    high_impact: bool
    listed: int
    listings: list[str]
    sources: int
    via: DeliverabilityAssetBlocklistsVia
    checked_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        high_impact = self.high_impact

        listed = self.listed

        listings = self.listings



        sources = self.sources

        via = self.via.value

        checked_at: str | Unset = UNSET
        if not isinstance(self.checked_at, Unset):
            checked_at = self.checked_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "highImpact": high_impact,
            "listed": listed,
            "listings": listings,
            "sources": sources,
            "via": via,
        })
        if checked_at is not UNSET:
            field_dict["checkedAt"] = checked_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        high_impact = d.pop("highImpact")

        listed = d.pop("listed")

        listings = cast(list[str], d.pop("listings"))


        sources = d.pop("sources")

        via = DeliverabilityAssetBlocklistsVia(d.pop("via"))




        _checked_at = d.pop("checkedAt", UNSET)
        checked_at: datetime.datetime | Unset
        if isinstance(_checked_at,  Unset):
            checked_at = UNSET
        else:
            checked_at = datetime.datetime.fromisoformat(_checked_at)




        deliverability_asset_blocklists = cls(
            high_impact=high_impact,
            listed=listed,
            listings=listings,
            sources=sources,
            via=via,
            checked_at=checked_at,
        )

        return deliverability_asset_blocklists

