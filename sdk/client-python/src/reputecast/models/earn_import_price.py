from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.earn_import_price_period import EarnImportPricePeriod
from ..types import UNSET, Unset






T = TypeVar("T", bound="EarnImportPrice")



@_attrs_define
class EarnImportPrice:
    """ 
        Attributes:
            amount_cents (int):
            currency (str):
            known (bool):
            legacy (bool):
            period (EarnImportPricePeriod):
            price_id (str):
            subscriptions (int):
            tier_id (str | Unset):
     """

    amount_cents: int
    currency: str
    known: bool
    legacy: bool
    period: EarnImportPricePeriod
    price_id: str
    subscriptions: int
    tier_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        amount_cents = self.amount_cents

        currency = self.currency

        known = self.known

        legacy = self.legacy

        period = self.period.value

        price_id = self.price_id

        subscriptions = self.subscriptions

        tier_id = self.tier_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "amountCents": amount_cents,
            "currency": currency,
            "known": known,
            "legacy": legacy,
            "period": period,
            "priceId": price_id,
            "subscriptions": subscriptions,
        })
        if tier_id is not UNSET:
            field_dict["tierId"] = tier_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        amount_cents = d.pop("amountCents")

        currency = d.pop("currency")

        known = d.pop("known")

        legacy = d.pop("legacy")

        period = EarnImportPricePeriod(d.pop("period"))




        price_id = d.pop("priceId")

        subscriptions = d.pop("subscriptions")

        tier_id = d.pop("tierId", UNSET)

        earn_import_price = cls(
            amount_cents=amount_cents,
            currency=currency,
            known=known,
            legacy=legacy,
            period=period,
            price_id=price_id,
            subscriptions=subscriptions,
            tier_id=tier_id,
        )

        return earn_import_price

