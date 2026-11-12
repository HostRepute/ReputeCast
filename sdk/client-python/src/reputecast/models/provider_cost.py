from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ProviderCost")



@_attrs_define
class ProviderCost:
    """ 
        Attributes:
            cost (float):
            provider (str):
            short (str):
     """

    cost: float
    provider: str
    short: str





    def to_dict(self) -> dict[str, Any]:
        cost = self.cost

        provider = self.provider

        short = self.short


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "cost": cost,
            "provider": provider,
            "short": short,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        cost = d.pop("cost")

        provider = d.pop("provider")

        short = d.pop("short")

        provider_cost = cls(
            cost=cost,
            provider=provider,
            short=short,
        )

        return provider_cost

