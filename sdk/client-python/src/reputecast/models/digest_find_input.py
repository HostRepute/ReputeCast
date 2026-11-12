from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="DigestFindInput")



@_attrs_define
class DigestFindInput:
    """ 
        Attributes:
            address (str):
     """

    address: str





    def to_dict(self) -> dict[str, Any]:
        address = self.address


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "address": address,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        address = d.pop("address")

        digest_find_input = cls(
            address=address,
        )

        return digest_find_input

