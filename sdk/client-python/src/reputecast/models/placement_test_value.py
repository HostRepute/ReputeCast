from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PlacementTestValue")



@_attrs_define
class PlacementTestValue:
    """ 
        Attributes:
            name (str):
            value (str):
     """

    name: str
    value: str





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        value = self.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
            "value": value,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        value = d.pop("value")

        placement_test_value = cls(
            name=name,
            value=value,
        )

        return placement_test_value

