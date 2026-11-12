from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WhatsAppReportCountry")



@_attrs_define
class WhatsAppReportCountry:
    """ 
        Attributes:
            cost (float):
            country (str):
            delivered (int):
            failed (int):
            read (int):
            sent (int):
     """

    cost: float
    country: str
    delivered: int
    failed: int
    read: int
    sent: int





    def to_dict(self) -> dict[str, Any]:
        cost = self.cost

        country = self.country

        delivered = self.delivered

        failed = self.failed

        read = self.read

        sent = self.sent


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "cost": cost,
            "country": country,
            "delivered": delivered,
            "failed": failed,
            "read": read,
            "sent": sent,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        cost = d.pop("cost")

        country = d.pop("country")

        delivered = d.pop("delivered")

        failed = d.pop("failed")

        read = d.pop("read")

        sent = d.pop("sent")

        whats_app_report_country = cls(
            cost=cost,
            country=country,
            delivered=delivered,
            failed=failed,
            read=read,
            sent=sent,
        )

        return whats_app_report_country

