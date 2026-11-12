from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="HostReputeAddAssetInputBody")



@_attrs_define
class HostReputeAddAssetInputBody:
    """ 
        Attributes:
            asset (str): A sending domain, organization domain, dedicated IP or server hostname of this workspace
     """

    asset: str





    def to_dict(self) -> dict[str, Any]:
        asset = self.asset


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "asset": asset,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        asset = d.pop("asset")

        host_repute_add_asset_input_body = cls(
            asset=asset,
        )

        return host_repute_add_asset_input_body

