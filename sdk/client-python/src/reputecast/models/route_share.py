from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="RouteShare")



@_attrs_define
class RouteShare:
    """ 
        Attributes:
            adapter (str):  Example: ses.
            percent (float):
            planned (int):
            provider (str):  Example: Amazon SES.
            provider_id (str):
            sent (int):
            short (str):  Example: SES.
     """

    adapter: str
    percent: float
    planned: int
    provider: str
    provider_id: str
    sent: int
    short: str





    def to_dict(self) -> dict[str, Any]:
        adapter = self.adapter

        percent = self.percent

        planned = self.planned

        provider = self.provider

        provider_id = self.provider_id

        sent = self.sent

        short = self.short


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "percent": percent,
            "planned": planned,
            "provider": provider,
            "providerId": provider_id,
            "sent": sent,
            "short": short,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        adapter = d.pop("adapter")

        percent = d.pop("percent")

        planned = d.pop("planned")

        provider = d.pop("provider")

        provider_id = d.pop("providerId")

        sent = d.pop("sent")

        short = d.pop("short")

        route_share = cls(
            adapter=adapter,
            percent=percent,
            planned=planned,
            provider=provider,
            provider_id=provider_id,
            sent=sent,
            short=short,
        )

        return route_share

