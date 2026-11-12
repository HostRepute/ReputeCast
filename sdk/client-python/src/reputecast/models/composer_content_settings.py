from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="ComposerContentSettings")



@_attrs_define
class ComposerContentSettings:
    """ 
        Attributes:
            minify (bool):
            utm (bool | Unset): Add UTM tags to the campaign's links; omit to follow the workspace's setting
     """

    minify: bool
    utm: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        minify = self.minify

        utm = self.utm


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "minify": minify,
        })
        if utm is not UNSET:
            field_dict["utm"] = utm

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        minify = d.pop("minify")

        utm = d.pop("utm", UNSET)

        composer_content_settings = cls(
            minify=minify,
            utm=utm,
        )

        return composer_content_settings

