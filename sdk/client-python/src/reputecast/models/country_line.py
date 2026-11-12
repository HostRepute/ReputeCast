from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="CountryLine")



@_attrs_define
class CountryLine:
    """ 
        Attributes:
            adapter (str):
            cost (float):
            country (str):
            pending_price (float):
            price (float):
            provider (str):
            recipients (int):
            segments (int):
            sender (str):
            sender_kind (str):
            skipped (int):
            skip_reason (str | Unset):
     """

    adapter: str
    cost: float
    country: str
    pending_price: float
    price: float
    provider: str
    recipients: int
    segments: int
    sender: str
    sender_kind: str
    skipped: int
    skip_reason: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        adapter = self.adapter

        cost = self.cost

        country = self.country

        pending_price = self.pending_price

        price = self.price

        provider = self.provider

        recipients = self.recipients

        segments = self.segments

        sender = self.sender

        sender_kind = self.sender_kind

        skipped = self.skipped

        skip_reason = self.skip_reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "cost": cost,
            "country": country,
            "pendingPrice": pending_price,
            "price": price,
            "provider": provider,
            "recipients": recipients,
            "segments": segments,
            "sender": sender,
            "senderKind": sender_kind,
            "skipped": skipped,
        })
        if skip_reason is not UNSET:
            field_dict["skipReason"] = skip_reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        adapter = d.pop("adapter")

        cost = d.pop("cost")

        country = d.pop("country")

        pending_price = d.pop("pendingPrice")

        price = d.pop("price")

        provider = d.pop("provider")

        recipients = d.pop("recipients")

        segments = d.pop("segments")

        sender = d.pop("sender")

        sender_kind = d.pop("senderKind")

        skipped = d.pop("skipped")

        skip_reason = d.pop("skipReason", UNSET)

        country_line = cls(
            adapter=adapter,
            cost=cost,
            country=country,
            pending_price=pending_price,
            price=price,
            provider=provider,
            recipients=recipients,
            segments=segments,
            sender=sender,
            sender_kind=sender_kind,
            skipped=skipped,
            skip_reason=skip_reason,
        )

        return country_line

