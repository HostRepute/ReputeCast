from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="CampaignConversions")



@_attrs_define
class CampaignConversions:
    """ 
        Attributes:
            by_email (int):
            by_token (int):
            count (int):
            currency (str):
            domains (list[str]):
            value_cents (int):
            name (str | Unset):
            sender (str | Unset):
            typical_cents (int | Unset):
     """

    by_email: int
    by_token: int
    count: int
    currency: str
    domains: list[str]
    value_cents: int
    name: str | Unset = UNSET
    sender: str | Unset = UNSET
    typical_cents: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        by_email = self.by_email

        by_token = self.by_token

        count = self.count

        currency = self.currency

        domains = self.domains



        value_cents = self.value_cents

        name = self.name

        sender = self.sender

        typical_cents = self.typical_cents


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "byEmail": by_email,
            "byToken": by_token,
            "count": count,
            "currency": currency,
            "domains": domains,
            "valueCents": value_cents,
        })
        if name is not UNSET:
            field_dict["name"] = name
        if sender is not UNSET:
            field_dict["sender"] = sender
        if typical_cents is not UNSET:
            field_dict["typicalCents"] = typical_cents

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        by_email = d.pop("byEmail")

        by_token = d.pop("byToken")

        count = d.pop("count")

        currency = d.pop("currency")

        domains = cast(list[str], d.pop("domains"))


        value_cents = d.pop("valueCents")

        name = d.pop("name", UNSET)

        sender = d.pop("sender", UNSET)

        typical_cents = d.pop("typicalCents", UNSET)

        campaign_conversions = cls(
            by_email=by_email,
            by_token=by_token,
            count=count,
            currency=currency,
            domains=domains,
            value_cents=value_cents,
            name=name,
            sender=sender,
            typical_cents=typical_cents,
        )

        return campaign_conversions

