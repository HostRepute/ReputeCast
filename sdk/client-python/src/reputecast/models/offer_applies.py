from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.offer_applies_period import OfferAppliesPeriod






T = TypeVar("T", bound="OfferApplies")



@_attrs_define
class OfferApplies:
    """ 
        Attributes:
            currency (str):
            period (OfferAppliesPeriod):
            price_cents (int):
            short (str):
            tier (str):
            tier_id (str):
     """

    currency: str
    period: OfferAppliesPeriod
    price_cents: int
    short: str
    tier: str
    tier_id: str





    def to_dict(self) -> dict[str, Any]:
        currency = self.currency

        period = self.period.value

        price_cents = self.price_cents

        short = self.short

        tier = self.tier

        tier_id = self.tier_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "currency": currency,
            "period": period,
            "priceCents": price_cents,
            "short": short,
            "tier": tier,
            "tierId": tier_id,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        currency = d.pop("currency")

        period = OfferAppliesPeriod(d.pop("period"))




        price_cents = d.pop("priceCents")

        short = d.pop("short")

        tier = d.pop("tier")

        tier_id = d.pop("tierId")

        offer_applies = cls(
            currency=currency,
            period=period,
            price_cents=price_cents,
            short=short,
            tier=tier,
            tier_id=tier_id,
        )

        return offer_applies

