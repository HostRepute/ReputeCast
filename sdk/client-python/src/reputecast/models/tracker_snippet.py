from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="TrackerSnippet")



@_attrs_define
class TrackerSnippet:
    """ 
        Attributes:
            script (str):
            shopify_pixel (str):
            wordpress (str):
     """

    script: str
    shopify_pixel: str
    wordpress: str





    def to_dict(self) -> dict[str, Any]:
        script = self.script

        shopify_pixel = self.shopify_pixel

        wordpress = self.wordpress


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "script": script,
            "shopifyPixel": shopify_pixel,
            "wordpress": wordpress,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        script = d.pop("script")

        shopify_pixel = d.pop("shopifyPixel")

        wordpress = d.pop("wordpress")

        tracker_snippet = cls(
            script=script,
            shopify_pixel=shopify_pixel,
            wordpress=wordpress,
        )

        return tracker_snippet

