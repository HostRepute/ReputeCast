from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.revenue_tier_offer import RevenueTierOffer





T = TypeVar("T", bound="RevenueTierRow")



@_attrs_define
class RevenueTierRow:
    """ 
        Attributes:
            includes (list[str]):
            joined (int):
            left (int):
            members (int):
            mrr_cents (int):
            offers (list[RevenueTierOffer]):
            period (str):
            price_cents (int):
            tier (str):
            tier_id (str):
     """

    includes: list[str]
    joined: int
    left: int
    members: int
    mrr_cents: int
    offers: list[RevenueTierOffer]
    period: str
    price_cents: int
    tier: str
    tier_id: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.revenue_tier_offer import RevenueTierOffer # noqa: PLC0415
        includes = self.includes



        joined = self.joined

        left = self.left

        members = self.members

        mrr_cents = self.mrr_cents

        offers = []
        for offers_item_data in self.offers:
            offers_item = offers_item_data.to_dict()
            offers.append(offers_item)



        period = self.period

        price_cents = self.price_cents

        tier = self.tier

        tier_id = self.tier_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "includes": includes,
            "joined": joined,
            "left": left,
            "members": members,
            "mrrCents": mrr_cents,
            "offers": offers,
            "period": period,
            "priceCents": price_cents,
            "tier": tier,
            "tierId": tier_id,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.revenue_tier_offer import RevenueTierOffer # noqa: PLC0415
        d = dict(src_dict)
        includes = cast(list[str], d.pop("includes"))


        joined = d.pop("joined")

        left = d.pop("left")

        members = d.pop("members")

        mrr_cents = d.pop("mrrCents")

        offers = []
        _offers = d.pop("offers")
        for offers_item_data in (_offers):
            offers_item = RevenueTierOffer.from_dict(offers_item_data)



            offers.append(offers_item)


        period = d.pop("period")

        price_cents = d.pop("priceCents")

        tier = d.pop("tier")

        tier_id = d.pop("tierId")

        revenue_tier_row = cls(
            includes=includes,
            joined=joined,
            left=left,
            members=members,
            mrr_cents=mrr_cents,
            offers=offers,
            period=period,
            price_cents=price_cents,
            tier=tier,
            tier_id=tier_id,
        )

        return revenue_tier_row

