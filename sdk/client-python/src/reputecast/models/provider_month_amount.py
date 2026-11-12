from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ProviderMonthAmount")



@_attrs_define
class ProviderMonthAmount:
    """ 
        Attributes:
            provider_id (str):
            short (str):
            value (float):
     """

    provider_id: str
    short: str
    value: float





    def to_dict(self) -> dict[str, Any]:
        provider_id = self.provider_id

        short = self.short

        value = self.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "providerId": provider_id,
            "short": short,
            "value": value,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        provider_id = d.pop("providerId")

        short = d.pop("short")

        value = d.pop("value")

        provider_month_amount = cls(
            provider_id=provider_id,
            short=short,
            value=value,
        )

        return provider_month_amount

