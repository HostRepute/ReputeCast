from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="SmsSenderReport")



@_attrs_define
class SmsSenderReport:
    """ 
        Attributes:
            adapter (str):
            cost (float):
            countries (list[str]):
            delivered (int):
            delivered_pct (float):
            failed (int):
            provider (str):
            sender (str):
            sender_kind (str):
            sent (int):
            throughput (float):
            zone_seconds (float):
     """

    adapter: str
    cost: float
    countries: list[str]
    delivered: int
    delivered_pct: float
    failed: int
    provider: str
    sender: str
    sender_kind: str
    sent: int
    throughput: float
    zone_seconds: float





    def to_dict(self) -> dict[str, Any]:
        adapter = self.adapter

        cost = self.cost

        countries = self.countries



        delivered = self.delivered

        delivered_pct = self.delivered_pct

        failed = self.failed

        provider = self.provider

        sender = self.sender

        sender_kind = self.sender_kind

        sent = self.sent

        throughput = self.throughput

        zone_seconds = self.zone_seconds


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "cost": cost,
            "countries": countries,
            "delivered": delivered,
            "deliveredPct": delivered_pct,
            "failed": failed,
            "provider": provider,
            "sender": sender,
            "senderKind": sender_kind,
            "sent": sent,
            "throughput": throughput,
            "zoneSeconds": zone_seconds,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        adapter = d.pop("adapter")

        cost = d.pop("cost")

        countries = cast(list[str], d.pop("countries"))


        delivered = d.pop("delivered")

        delivered_pct = d.pop("deliveredPct")

        failed = d.pop("failed")

        provider = d.pop("provider")

        sender = d.pop("sender")

        sender_kind = d.pop("senderKind")

        sent = d.pop("sent")

        throughput = d.pop("throughput")

        zone_seconds = d.pop("zoneSeconds")

        sms_sender_report = cls(
            adapter=adapter,
            cost=cost,
            countries=countries,
            delivered=delivered,
            delivered_pct=delivered_pct,
            failed=failed,
            provider=provider,
            sender=sender,
            sender_kind=sender_kind,
            sent=sent,
            throughput=throughput,
            zone_seconds=zone_seconds,
        )

        return sms_sender_report

