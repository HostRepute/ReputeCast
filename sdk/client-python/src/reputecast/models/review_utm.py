from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReviewUTM")



@_attrs_define
class ReviewUTM:
    """ 
        Attributes:
            campaign (str):
            medium (str):
            source (str):
     """

    campaign: str
    medium: str
    source: str





    def to_dict(self) -> dict[str, Any]:
        campaign = self.campaign

        medium = self.medium

        source = self.source


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "campaign": campaign,
            "medium": medium,
            "source": source,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        campaign = d.pop("campaign")

        medium = d.pop("medium")

        source = d.pop("source")

        review_utm = cls(
            campaign=campaign,
            medium=medium,
            source=source,
        )

        return review_utm

