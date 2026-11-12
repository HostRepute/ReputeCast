from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.public_branding_input_font import PublicBrandingInputFont
from ..types import UNSET, Unset






T = TypeVar("T", bound="PublicBrandingInput")



@_attrs_define
class PublicBrandingInput:
    """ 
        Attributes:
            accent (str):  Example: #4338CA.
            background (str):  Example: #F5F3FF.
            font (PublicBrandingInputFont):
            powered_by (bool):
            publisher_name (str):
            custom_font_family (str | Unset):
            custom_font_url (str | Unset): https URL of a .woff2 or .woff file
            privacy_url (str | Unset):
     """

    accent: str
    background: str
    font: PublicBrandingInputFont
    powered_by: bool
    publisher_name: str
    custom_font_family: str | Unset = UNSET
    custom_font_url: str | Unset = UNSET
    privacy_url: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        accent = self.accent

        background = self.background

        font = self.font.value

        powered_by = self.powered_by

        publisher_name = self.publisher_name

        custom_font_family = self.custom_font_family

        custom_font_url = self.custom_font_url

        privacy_url = self.privacy_url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "accent": accent,
            "background": background,
            "font": font,
            "poweredBy": powered_by,
            "publisherName": publisher_name,
        })
        if custom_font_family is not UNSET:
            field_dict["customFontFamily"] = custom_font_family
        if custom_font_url is not UNSET:
            field_dict["customFontUrl"] = custom_font_url
        if privacy_url is not UNSET:
            field_dict["privacyUrl"] = privacy_url

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        accent = d.pop("accent")

        background = d.pop("background")

        font = PublicBrandingInputFont(d.pop("font"))




        powered_by = d.pop("poweredBy")

        publisher_name = d.pop("publisherName")

        custom_font_family = d.pop("customFontFamily", UNSET)

        custom_font_url = d.pop("customFontUrl", UNSET)

        privacy_url = d.pop("privacyUrl", UNSET)

        public_branding_input = cls(
            accent=accent,
            background=background,
            font=font,
            powered_by=powered_by,
            publisher_name=publisher_name,
            custom_font_family=custom_font_family,
            custom_font_url=custom_font_url,
            privacy_url=privacy_url,
        )

        return public_branding_input

