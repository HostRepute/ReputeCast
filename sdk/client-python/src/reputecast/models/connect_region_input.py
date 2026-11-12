from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ConnectRegionInput")



@_attrs_define
class ConnectRegionInput:
    """ 
        Attributes:
            region (str):
     """

    region: str





    def to_dict(self) -> dict[str, Any]:
        region = self.region


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "region": region,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        region = d.pop("region")

        connect_region_input = cls(
            region=region,
        )

        return connect_region_input

