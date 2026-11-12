from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WarmupCampaignUse")



@_attrs_define
class WarmupCampaignUse:
    """ 
        Attributes:
            count (int):
            name (str):
     """

    count: int
    name: str





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        name = d.pop("name")

        warmup_campaign_use = cls(
            count=count,
            name=name,
        )

        return warmup_campaign_use

