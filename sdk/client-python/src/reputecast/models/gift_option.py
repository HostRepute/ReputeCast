from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="GiftOption")



@_attrs_define
class GiftOption:
    """ 
        Attributes:
            amount_cents (int):
            currency (str):
            id (str):
            months (int):
            short (str):
            sold (int):
            sold_cents (int):
            tier (str):
            tier_id (str):
     """

    amount_cents: int
    currency: str
    id: str
    months: int
    short: str
    sold: int
    sold_cents: int
    tier: str
    tier_id: str





    def to_dict(self) -> dict[str, Any]:
        amount_cents = self.amount_cents

        currency = self.currency

        id = self.id

        months = self.months

        short = self.short

        sold = self.sold

        sold_cents = self.sold_cents

        tier = self.tier

        tier_id = self.tier_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "amountCents": amount_cents,
            "currency": currency,
            "id": id,
            "months": months,
            "short": short,
            "sold": sold,
            "soldCents": sold_cents,
            "tier": tier,
            "tierId": tier_id,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        amount_cents = d.pop("amountCents")

        currency = d.pop("currency")

        id = d.pop("id")

        months = d.pop("months")

        short = d.pop("short")

        sold = d.pop("sold")

        sold_cents = d.pop("soldCents")

        tier = d.pop("tier")

        tier_id = d.pop("tierId")

        gift_option = cls(
            amount_cents=amount_cents,
            currency=currency,
            id=id,
            months=months,
            short=short,
            sold=sold,
            sold_cents=sold_cents,
            tier=tier,
            tier_id=tier_id,
        )

        return gift_option

