from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sponsor_report_link_kind import SponsorReportLinkKind






T = TypeVar("T", bound="SponsorReportLink")



@_attrs_define
class SponsorReportLink:
    """ 
        Attributes:
            clicks (int):
            kind (SponsorReportLinkKind):
            label (str):
            share (float):
     """

    clicks: int
    kind: SponsorReportLinkKind
    label: str
    share: float





    def to_dict(self) -> dict[str, Any]:
        clicks = self.clicks

        kind = self.kind.value

        label = self.label

        share = self.share


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "clicks": clicks,
            "kind": kind,
            "label": label,
            "share": share,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        clicks = d.pop("clicks")

        kind = SponsorReportLinkKind(d.pop("kind"))




        label = d.pop("label")

        share = d.pop("share")

        sponsor_report_link = cls(
            clicks=clicks,
            kind=kind,
            label=label,
            share=share,
        )

        return sponsor_report_link

