from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="CampaignPayment")



@_attrs_define
class CampaignPayment:
    """ 
        Attributes:
            amount_cents (int):
            count (int):
     """

    amount_cents: int
    count: int





    def to_dict(self) -> dict[str, Any]:
        amount_cents = self.amount_cents

        count = self.count


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "amountCents": amount_cents,
            "count": count,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        amount_cents = d.pop("amountCents")

        count = d.pop("count")

        campaign_payment = cls(
            amount_cents=amount_cents,
            count=count,
        )

        return campaign_payment

