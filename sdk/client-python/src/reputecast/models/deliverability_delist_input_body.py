from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="DeliverabilityDelistInputBody")



@_attrs_define
class DeliverabilityDelistInputBody:
    """ 
        Attributes:
            asset (str):
            source (str | Unset): One blocklist; every list the asset is on when omitted
     """

    asset: str
    source: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        asset = self.asset

        source = self.source


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "asset": asset,
        })
        if source is not UNSET:
            field_dict["source"] = source

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        asset = d.pop("asset")

        source = d.pop("source", UNSET)

        deliverability_delist_input_body = cls(
            asset=asset,
            source=source,
        )

        return deliverability_delist_input_body

