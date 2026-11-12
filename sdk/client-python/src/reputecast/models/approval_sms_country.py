from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ApprovalSMSCountry")



@_attrs_define
class ApprovalSMSCountry:
    """ 
        Attributes:
            cost (float):
            country (str):
            price (float):
            provider (str):
            recipients (int):
            sender (str):
            skipped (bool):
     """

    cost: float
    country: str
    price: float
    provider: str
    recipients: int
    sender: str
    skipped: bool





    def to_dict(self) -> dict[str, Any]:
        cost = self.cost

        country = self.country

        price = self.price

        provider = self.provider

        recipients = self.recipients

        sender = self.sender

        skipped = self.skipped


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "cost": cost,
            "country": country,
            "price": price,
            "provider": provider,
            "recipients": recipients,
            "sender": sender,
            "skipped": skipped,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        cost = d.pop("cost")

        country = d.pop("country")

        price = d.pop("price")

        provider = d.pop("provider")

        recipients = d.pop("recipients")

        sender = d.pop("sender")

        skipped = d.pop("skipped")

        approval_sms_country = cls(
            cost=cost,
            country=country,
            price=price,
            provider=provider,
            recipients=recipients,
            sender=sender,
            skipped=skipped,
        )

        return approval_sms_country

