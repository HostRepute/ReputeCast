from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="AutomationProviderSends")



@_attrs_define
class AutomationProviderSends:
    """ 
        Attributes:
            fallback (bool):
            provider (str):
            provider_id (str):
            sent (int):
            short (str):
            last_day (datetime.date | Unset):  Example: 2026-10-02.
     """

    fallback: bool
    provider: str
    provider_id: str
    sent: int
    short: str
    last_day: datetime.date | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        fallback = self.fallback

        provider = self.provider

        provider_id = self.provider_id

        sent = self.sent

        short = self.short

        last_day: str | Unset = UNSET
        if not isinstance(self.last_day, Unset):
            last_day = self.last_day.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "fallback": fallback,
            "provider": provider,
            "providerId": provider_id,
            "sent": sent,
            "short": short,
        })
        if last_day is not UNSET:
            field_dict["lastDay"] = last_day

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        fallback = d.pop("fallback")

        provider = d.pop("provider")

        provider_id = d.pop("providerId")

        sent = d.pop("sent")

        short = d.pop("short")

        _last_day = d.pop("lastDay", UNSET)
        last_day: datetime.date | Unset
        if isinstance(_last_day,  Unset):
            last_day = UNSET
        else:
            last_day = datetime.date.fromisoformat(_last_day)




        automation_provider_sends = cls(
            fallback=fallback,
            provider=provider,
            provider_id=provider_id,
            sent=sent,
            short=short,
            last_day=last_day,
        )

        return automation_provider_sends

