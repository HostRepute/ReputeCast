from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WebsiteSource")



@_attrs_define
class WebsiteSource:
    """ 
        Attributes:
            key (str):
            label (str):
            views (int):
     """

    key: str
    label: str
    views: int





    def to_dict(self) -> dict[str, Any]:
        key = self.key

        label = self.label

        views = self.views


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "label": label,
            "views": views,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = d.pop("key")

        label = d.pop("label")

        views = d.pop("views")

        website_source = cls(
            key=key,
            label=label,
            views=views,
        )

        return website_source

