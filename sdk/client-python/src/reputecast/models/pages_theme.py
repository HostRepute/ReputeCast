from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PagesTheme")



@_attrs_define
class PagesTheme:
    """ 
        Attributes:
            accent (str):
            font (str):
            ground (str):
            logo (bool):
            publisher (str):
     """

    accent: str
    font: str
    ground: str
    logo: bool
    publisher: str





    def to_dict(self) -> dict[str, Any]:
        accent = self.accent

        font = self.font

        ground = self.ground

        logo = self.logo

        publisher = self.publisher


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "accent": accent,
            "font": font,
            "ground": ground,
            "logo": logo,
            "publisher": publisher,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        accent = d.pop("accent")

        font = d.pop("font")

        ground = d.pop("ground")

        logo = d.pop("logo")

        publisher = d.pop("publisher")

        pages_theme = cls(
            accent=accent,
            font=font,
            ground=ground,
            logo=logo,
            publisher=publisher,
        )

        return pages_theme

