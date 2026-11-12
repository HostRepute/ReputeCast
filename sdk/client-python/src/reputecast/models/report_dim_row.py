from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="ReportDimRow")



@_attrs_define
class ReportDimRow:
    """ 
        Attributes:
            clicks (int):
            human_opens (int):
            key (str):
            label (str):
            share (float):
            country (str | Unset):
            places (int | Unset):
     """

    clicks: int
    human_opens: int
    key: str
    label: str
    share: float
    country: str | Unset = UNSET
    places: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        clicks = self.clicks

        human_opens = self.human_opens

        key = self.key

        label = self.label

        share = self.share

        country = self.country

        places = self.places


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "clicks": clicks,
            "humanOpens": human_opens,
            "key": key,
            "label": label,
            "share": share,
        })
        if country is not UNSET:
            field_dict["country"] = country
        if places is not UNSET:
            field_dict["places"] = places

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        clicks = d.pop("clicks")

        human_opens = d.pop("humanOpens")

        key = d.pop("key")

        label = d.pop("label")

        share = d.pop("share")

        country = d.pop("country", UNSET)

        places = d.pop("places", UNSET)

        report_dim_row = cls(
            clicks=clicks,
            human_opens=human_opens,
            key=key,
            label=label,
            share=share,
            country=country,
            places=places,
        )

        return report_dim_row

