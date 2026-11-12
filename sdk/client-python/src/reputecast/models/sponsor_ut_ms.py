from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="SponsorUTMs")



@_attrs_define
class SponsorUTMs:
    """ 
        Attributes:
            campaign (str | Unset):
            medium (str | Unset):
            source (str | Unset):
     """

    campaign: str | Unset = UNSET
    medium: str | Unset = UNSET
    source: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        campaign = self.campaign

        medium = self.medium

        source = self.source


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if campaign is not UNSET:
            field_dict["campaign"] = campaign
        if medium is not UNSET:
            field_dict["medium"] = medium
        if source is not UNSET:
            field_dict["source"] = source

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        campaign = d.pop("campaign", UNSET)

        medium = d.pop("medium", UNSET)

        source = d.pop("source", UNSET)

        sponsor_ut_ms = cls(
            campaign=campaign,
            medium=medium,
            source=source,
        )

        return sponsor_ut_ms

