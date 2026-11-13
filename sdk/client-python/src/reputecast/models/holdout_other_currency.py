from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="HoldoutOtherCurrency")



@_attrs_define
class HoldoutOtherCurrency:
    """ 
        Attributes:
            currency (str):
            holdout_cents (int):
            marketing_cents (int):
     """

    currency: str
    holdout_cents: int
    marketing_cents: int





    def to_dict(self) -> dict[str, Any]:
        currency = self.currency

        holdout_cents = self.holdout_cents

        marketing_cents = self.marketing_cents


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "currency": currency,
            "holdoutCents": holdout_cents,
            "marketingCents": marketing_cents,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        currency = d.pop("currency")

        holdout_cents = d.pop("holdoutCents")

        marketing_cents = d.pop("marketingCents")

        holdout_other_currency = cls(
            currency=currency,
            holdout_cents=holdout_cents,
            marketing_cents=marketing_cents,
        )

        return holdout_other_currency

