from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="RevenueTierOffer")



@_attrs_define
class RevenueTierOffer:
    """ 
        Attributes:
            duration (str):
            members (int):
            name (str):
            paid_cents (int):
            months (int | Unset):
     """

    duration: str
    members: int
    name: str
    paid_cents: int
    months: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        duration = self.duration

        members = self.members

        name = self.name

        paid_cents = self.paid_cents

        months = self.months


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "duration": duration,
            "members": members,
            "name": name,
            "paidCents": paid_cents,
        })
        if months is not UNSET:
            field_dict["months"] = months

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        duration = d.pop("duration")

        members = d.pop("members")

        name = d.pop("name")

        paid_cents = d.pop("paidCents")

        months = d.pop("months", UNSET)

        revenue_tier_offer = cls(
            duration=duration,
            members=members,
            name=name,
            paid_cents=paid_cents,
            months=months,
        )

        return revenue_tier_offer

