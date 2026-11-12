from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="ReportLinkGroup")



@_attrs_define
class ReportLinkGroup:
    """ 
        Attributes:
            clicks (int):
            labels (list[str]): The most-clicked links in the group, for the caption
            links (int):
            share (float):
     """

    clicks: int
    labels: list[str]
    links: int
    share: float





    def to_dict(self) -> dict[str, Any]:
        clicks = self.clicks

        labels = self.labels



        links = self.links

        share = self.share


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "clicks": clicks,
            "labels": labels,
            "links": links,
            "share": share,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        clicks = d.pop("clicks")

        labels = cast(list[str], d.pop("labels"))


        links = d.pop("links")

        share = d.pop("share")

        report_link_group = cls(
            clicks=clicks,
            labels=labels,
            links=links,
            share=share,
        )

        return report_link_group

