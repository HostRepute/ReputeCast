from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SmsReviewRoute")



@_attrs_define
class SmsReviewRoute:
    """ 
        Attributes:
            adapter (str):
            cost (float):
            provider (str):
            rate (float):
            recipients (int):
            seconds (float):
            sender (str):
            share (float):
     """

    adapter: str
    cost: float
    provider: str
    rate: float
    recipients: int
    seconds: float
    sender: str
    share: float





    def to_dict(self) -> dict[str, Any]:
        adapter = self.adapter

        cost = self.cost

        provider = self.provider

        rate = self.rate

        recipients = self.recipients

        seconds = self.seconds

        sender = self.sender

        share = self.share


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "cost": cost,
            "provider": provider,
            "rate": rate,
            "recipients": recipients,
            "seconds": seconds,
            "sender": sender,
            "share": share,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        adapter = d.pop("adapter")

        cost = d.pop("cost")

        provider = d.pop("provider")

        rate = d.pop("rate")

        recipients = d.pop("recipients")

        seconds = d.pop("seconds")

        sender = d.pop("sender")

        share = d.pop("share")

        sms_review_route = cls(
            adapter=adapter,
            cost=cost,
            provider=provider,
            rate=rate,
            recipients=recipients,
            seconds=seconds,
            sender=sender,
            share=share,
        )

        return sms_review_route

