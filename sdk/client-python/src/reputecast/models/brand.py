from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="Brand")



@_attrs_define
class Brand:
    """ 
        Attributes:
            accent (str):
            ground (str):
            on_accent (str):
            publisher (str):
            soft (str):
            text (str):
     """

    accent: str
    ground: str
    on_accent: str
    publisher: str
    soft: str
    text: str





    def to_dict(self) -> dict[str, Any]:
        accent = self.accent

        ground = self.ground

        on_accent = self.on_accent

        publisher = self.publisher

        soft = self.soft

        text = self.text


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "accent": accent,
            "ground": ground,
            "onAccent": on_accent,
            "publisher": publisher,
            "soft": soft,
            "text": text,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        accent = d.pop("accent")

        ground = d.pop("ground")

        on_accent = d.pop("onAccent")

        publisher = d.pop("publisher")

        soft = d.pop("soft")

        text = d.pop("text")

        brand = cls(
            accent=accent,
            ground=ground,
            on_accent=on_accent,
            publisher=publisher,
            soft=soft,
            text=text,
        )

        return brand

