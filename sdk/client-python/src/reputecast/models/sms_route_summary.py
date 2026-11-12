from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="SmsRouteSummary")



@_attrs_define
class SmsRouteSummary:
    """ 
        Attributes:
            countries (list[str]):
            fallback (str):
            fallback_provider (str):
            id (str):
            provider (str):
            registration (str):
            sender (str):
            skipped (bool):
     """

    countries: list[str]
    fallback: str
    fallback_provider: str
    id: str
    provider: str
    registration: str
    sender: str
    skipped: bool





    def to_dict(self) -> dict[str, Any]:
        countries = self.countries



        fallback = self.fallback

        fallback_provider = self.fallback_provider

        id = self.id

        provider = self.provider

        registration = self.registration

        sender = self.sender

        skipped = self.skipped


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "countries": countries,
            "fallback": fallback,
            "fallbackProvider": fallback_provider,
            "id": id,
            "provider": provider,
            "registration": registration,
            "sender": sender,
            "skipped": skipped,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        countries = cast(list[str], d.pop("countries"))


        fallback = d.pop("fallback")

        fallback_provider = d.pop("fallbackProvider")

        id = d.pop("id")

        provider = d.pop("provider")

        registration = d.pop("registration")

        sender = d.pop("sender")

        skipped = d.pop("skipped")

        sms_route_summary = cls(
            countries=countries,
            fallback=fallback,
            fallback_provider=fallback_provider,
            id=id,
            provider=provider,
            registration=registration,
            sender=sender,
            skipped=skipped,
        )

        return sms_route_summary

