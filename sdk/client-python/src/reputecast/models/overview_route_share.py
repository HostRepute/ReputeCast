from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.overview_route_share_color import OverviewRouteShareColor






T = TypeVar("T", bound="OverviewRouteShare")



@_attrs_define
class OverviewRouteShare:
    """ 
        Attributes:
            adapter (str):
            color (OverviewRouteShareColor):
            percent (float):
            planned (int):
            short (str):
     """

    adapter: str
    color: OverviewRouteShareColor
    percent: float
    planned: int
    short: str





    def to_dict(self) -> dict[str, Any]:
        adapter = self.adapter

        color = self.color.value

        percent = self.percent

        planned = self.planned

        short = self.short


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "color": color,
            "percent": percent,
            "planned": planned,
            "short": short,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        adapter = d.pop("adapter")

        color = OverviewRouteShareColor(d.pop("color"))




        percent = d.pop("percent")

        planned = d.pop("planned")

        short = d.pop("short")

        overview_route_share = cls(
            adapter=adapter,
            color=color,
            percent=percent,
            planned=planned,
            short=short,
        )

        return overview_route_share

