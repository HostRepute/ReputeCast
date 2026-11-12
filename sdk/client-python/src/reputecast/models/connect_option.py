from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ConnectOption")



@_attrs_define
class ConnectOption:
    """ 
        Attributes:
            label (str):
            value (str):
     """

    label: str
    value: str





    def to_dict(self) -> dict[str, Any]:
        label = self.label

        value = self.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "label": label,
            "value": value,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        label = d.pop("label")

        value = d.pop("value")

        connect_option = cls(
            label=label,
            value=value,
        )

        return connect_option

