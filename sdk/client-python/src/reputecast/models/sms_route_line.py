from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="SmsRouteLine")



@_attrs_define
class SmsRouteLine:
    """ 
        Attributes:
            adapter (str):
            countries (list[str]):
            provider (str):
            recipients (int):
            sender (str):
            skipped (str | Unset):
     """

    adapter: str
    countries: list[str]
    provider: str
    recipients: int
    sender: str
    skipped: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        adapter = self.adapter

        countries = self.countries



        provider = self.provider

        recipients = self.recipients

        sender = self.sender

        skipped = self.skipped


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "countries": countries,
            "provider": provider,
            "recipients": recipients,
            "sender": sender,
        })
        if skipped is not UNSET:
            field_dict["skipped"] = skipped

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        adapter = d.pop("adapter")

        countries = cast(list[str], d.pop("countries"))


        provider = d.pop("provider")

        recipients = d.pop("recipients")

        sender = d.pop("sender")

        skipped = d.pop("skipped", UNSET)

        sms_route_line = cls(
            adapter=adapter,
            countries=countries,
            provider=provider,
            recipients=recipients,
            sender=sender,
            skipped=skipped,
        )

        return sms_route_line

