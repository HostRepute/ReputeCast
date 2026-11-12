from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WhatsAppField")



@_attrs_define
class WhatsAppField:
    """ 
        Attributes:
            key (str):
            label (str):
     """

    key: str
    label: str





    def to_dict(self) -> dict[str, Any]:
        key = self.key

        label = self.label


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "label": label,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = d.pop("key")

        label = d.pop("label")

        whats_app_field = cls(
            key=key,
            label=label,
        )

        return whats_app_field

