from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReferralCheckFile")



@_attrs_define
class ReferralCheckFile:
    """ 
        Attributes:
            name (str):
            size (int):
     """

    name: str
    size: int





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        size = self.size


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
            "size": size,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        size = d.pop("size")

        referral_check_file = cls(
            name=name,
            size=size,
        )

        return referral_check_file

