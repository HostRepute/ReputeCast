from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.public_branding_font import PublicBrandingFont
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.public_branding_logo import PublicBrandingLogo





T = TypeVar("T", bound="PublicBranding")



@_attrs_define
class PublicBranding:
    """ 
        Attributes:
            accent (str):
            accent_hover (str):
            accent_soft (str):
            accent_text (str):
            background (str):
            custom_font_family (str):
            custom_font_url (str):
            font (PublicBrandingFont):
            initial (str): The letter on the logo tile when there is no logo
            notes (list[str]):
            on_accent (str):
            powered_by (bool): Show "Sent with ReputeCast" in the footer
            privacy_url (str):
            publisher_name (str):
            logo (PublicBrandingLogo | Unset):
     """

    accent: str
    accent_hover: str
    accent_soft: str
    accent_text: str
    background: str
    custom_font_family: str
    custom_font_url: str
    font: PublicBrandingFont
    initial: str
    notes: list[str]
    on_accent: str
    powered_by: bool
    privacy_url: str
    publisher_name: str
    logo: PublicBrandingLogo | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.public_branding_logo import PublicBrandingLogo # noqa: PLC0415
        accent = self.accent

        accent_hover = self.accent_hover

        accent_soft = self.accent_soft

        accent_text = self.accent_text

        background = self.background

        custom_font_family = self.custom_font_family

        custom_font_url = self.custom_font_url

        font = self.font.value

        initial = self.initial

        notes = self.notes



        on_accent = self.on_accent

        powered_by = self.powered_by

        privacy_url = self.privacy_url

        publisher_name = self.publisher_name

        logo: dict[str, Any] | Unset = UNSET
        if not isinstance(self.logo, Unset):
            logo = self.logo.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "accent": accent,
            "accentHover": accent_hover,
            "accentSoft": accent_soft,
            "accentText": accent_text,
            "background": background,
            "customFontFamily": custom_font_family,
            "customFontUrl": custom_font_url,
            "font": font,
            "initial": initial,
            "notes": notes,
            "onAccent": on_accent,
            "poweredBy": powered_by,
            "privacyUrl": privacy_url,
            "publisherName": publisher_name,
        })
        if logo is not UNSET:
            field_dict["logo"] = logo

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.public_branding_logo import PublicBrandingLogo # noqa: PLC0415
        d = dict(src_dict)
        accent = d.pop("accent")

        accent_hover = d.pop("accentHover")

        accent_soft = d.pop("accentSoft")

        accent_text = d.pop("accentText")

        background = d.pop("background")

        custom_font_family = d.pop("customFontFamily")

        custom_font_url = d.pop("customFontUrl")

        font = PublicBrandingFont(d.pop("font"))




        initial = d.pop("initial")

        notes = cast(list[str], d.pop("notes"))


        on_accent = d.pop("onAccent")

        powered_by = d.pop("poweredBy")

        privacy_url = d.pop("privacyUrl")

        publisher_name = d.pop("publisherName")

        _logo = d.pop("logo", UNSET)
        logo: PublicBrandingLogo | Unset
        if isinstance(_logo,  Unset):
            logo = UNSET
        else:
            logo = PublicBrandingLogo.from_dict(_logo)




        public_branding = cls(
            accent=accent,
            accent_hover=accent_hover,
            accent_soft=accent_soft,
            accent_text=accent_text,
            background=background,
            custom_font_family=custom_font_family,
            custom_font_url=custom_font_url,
            font=font,
            initial=initial,
            notes=notes,
            on_accent=on_accent,
            powered_by=powered_by,
            privacy_url=privacy_url,
            publisher_name=publisher_name,
            logo=logo,
        )

        return public_branding

