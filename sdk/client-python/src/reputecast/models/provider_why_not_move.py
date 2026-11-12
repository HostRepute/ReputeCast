from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ProviderWhyNotMove")



@_attrs_define
class ProviderWhyNotMove:
    """ 
        Attributes:
            age_days (int):
            held (int):
            multiple (float):
            provider_id (str):
            rate (float):
            today_cap (int):
     """

    age_days: int
    held: int
    multiple: float
    provider_id: str
    rate: float
    today_cap: int





    def to_dict(self) -> dict[str, Any]:
        age_days = self.age_days

        held = self.held

        multiple = self.multiple

        provider_id = self.provider_id

        rate = self.rate

        today_cap = self.today_cap


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "ageDays": age_days,
            "held": held,
            "multiple": multiple,
            "providerId": provider_id,
            "rate": rate,
            "todayCap": today_cap,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        age_days = d.pop("ageDays")

        held = d.pop("held")

        multiple = d.pop("multiple")

        provider_id = d.pop("providerId")

        rate = d.pop("rate")

        today_cap = d.pop("todayCap")

        provider_why_not_move = cls(
            age_days=age_days,
            held=held,
            multiple=multiple,
            provider_id=provider_id,
            rate=rate,
            today_cap=today_cap,
        )

        return provider_why_not_move

