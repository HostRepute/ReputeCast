from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PushButton")



@_attrs_define
class PushButton:
    """ 
        Attributes:
            label (str):
            url (str):
     """

    label: str
    url: str





    def to_dict(self) -> dict[str, Any]:
        label = self.label

        url = self.url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "label": label,
            "url": url,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        label = d.pop("label")

        url = d.pop("url")

        push_button = cls(
            label=label,
            url=url,
        )

        return push_button

