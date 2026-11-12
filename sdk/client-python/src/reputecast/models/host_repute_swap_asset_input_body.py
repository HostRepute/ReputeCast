from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="HostReputeSwapAssetInputBody")



@_attrs_define
class HostReputeSwapAssetInputBody:
    """ 
        Attributes:
            add (str):
            remove (str):
     """

    add: str
    remove: str





    def to_dict(self) -> dict[str, Any]:
        add = self.add

        remove = self.remove


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "add": add,
            "remove": remove,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        add = d.pop("add")

        remove = d.pop("remove")

        host_repute_swap_asset_input_body = cls(
            add=add,
            remove=remove,
        )

        return host_repute_swap_asset_input_body

