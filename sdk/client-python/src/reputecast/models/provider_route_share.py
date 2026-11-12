from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.provider_route_share_color import ProviderRouteShareColor






T = TypeVar("T", bound="ProviderRouteShare")



@_attrs_define
class ProviderRouteShare:
    """ 
        Attributes:
            color (ProviderRouteShareColor):
            percent (int):
            provider_id (str):
            short (str):
     """

    color: ProviderRouteShareColor
    percent: int
    provider_id: str
    short: str





    def to_dict(self) -> dict[str, Any]:
        color = self.color.value

        percent = self.percent

        provider_id = self.provider_id

        short = self.short


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "color": color,
            "percent": percent,
            "providerId": provider_id,
            "short": short,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        color = ProviderRouteShareColor(d.pop("color"))




        percent = d.pop("percent")

        provider_id = d.pop("providerId")

        short = d.pop("short")

        provider_route_share = cls(
            color=color,
            percent=percent,
            provider_id=provider_id,
            short=short,
        )

        return provider_route_share

