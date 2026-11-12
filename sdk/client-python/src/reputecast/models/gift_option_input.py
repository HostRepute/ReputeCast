from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from uuid import UUID






T = TypeVar("T", bound="GiftOptionInput")



@_attrs_define
class GiftOptionInput:
    """ 
        Attributes:
            amount_cents (int): The one-time price, in the workspace currency
            months (int): How long the gift lasts
            tier_id (UUID):
     """

    amount_cents: int
    months: int
    tier_id: UUID





    def to_dict(self) -> dict[str, Any]:
        amount_cents = self.amount_cents

        months = self.months

        tier_id = str(self.tier_id)


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "amountCents": amount_cents,
            "months": months,
            "tierId": tier_id,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        amount_cents = d.pop("amountCents")

        months = d.pop("months")

        tier_id = UUID(d.pop("tierId"))




        gift_option_input = cls(
            amount_cents=amount_cents,
            months=months,
            tier_id=tier_id,
        )

        return gift_option_input

