from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PlacementAuthCount")



@_attrs_define
class PlacementAuthCount:
    """ 
        Attributes:
            fail (int):
            pass_ (int):
     """

    fail: int
    pass_: int





    def to_dict(self) -> dict[str, Any]:
        fail = self.fail

        pass_ = self.pass_


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "fail": fail,
            "pass": pass_,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        fail = d.pop("fail")

        pass_ = d.pop("pass")

        placement_auth_count = cls(
            fail=fail,
            pass_=pass_,
        )

        return placement_auth_count

