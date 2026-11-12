from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.brand_import_color import BrandImportColor
  from ..models.brand_logo import BrandLogo
  from ..models.brand_style import BrandStyle





T = TypeVar("T", bound="BrandKitImport")



@_attrs_define
class BrandKitImport:
    """ 
        Attributes:
            colors (list[BrandImportColor]):
            fonts (list[str]):
            found_at (datetime.datetime):
            logo_alt (str):
            notes (list[str]):
            style (BrandStyle):
            website (str):
            logo_dark (BrandLogo | Unset):
            logo_light (BrandLogo | Unset):
     """

    colors: list[BrandImportColor]
    fonts: list[str]
    found_at: datetime.datetime
    logo_alt: str
    notes: list[str]
    style: BrandStyle
    website: str
    logo_dark: BrandLogo | Unset = UNSET
    logo_light: BrandLogo | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.brand_import_color import BrandImportColor # noqa: PLC0415
        from ..models.brand_logo import BrandLogo # noqa: PLC0415
        from ..models.brand_style import BrandStyle # noqa: PLC0415
        colors = []
        for colors_item_data in self.colors:
            colors_item = colors_item_data.to_dict()
            colors.append(colors_item)



        fonts = self.fonts



        found_at = self.found_at.isoformat()

        logo_alt = self.logo_alt

        notes = self.notes



        style = self.style.to_dict()

        website = self.website

        logo_dark: dict[str, Any] | Unset = UNSET
        if not isinstance(self.logo_dark, Unset):
            logo_dark = self.logo_dark.to_dict()

        logo_light: dict[str, Any] | Unset = UNSET
        if not isinstance(self.logo_light, Unset):
            logo_light = self.logo_light.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "colors": colors,
            "fonts": fonts,
            "foundAt": found_at,
            "logoAlt": logo_alt,
            "notes": notes,
            "style": style,
            "website": website,
        })
        if logo_dark is not UNSET:
            field_dict["logoDark"] = logo_dark
        if logo_light is not UNSET:
            field_dict["logoLight"] = logo_light

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.brand_import_color import BrandImportColor # noqa: PLC0415
        from ..models.brand_logo import BrandLogo # noqa: PLC0415
        from ..models.brand_style import BrandStyle # noqa: PLC0415
        d = dict(src_dict)
        colors = []
        _colors = d.pop("colors")
        for colors_item_data in (_colors):
            colors_item = BrandImportColor.from_dict(colors_item_data)



            colors.append(colors_item)


        fonts = cast(list[str], d.pop("fonts"))


        found_at = datetime.datetime.fromisoformat(d.pop("foundAt"))




        logo_alt = d.pop("logoAlt")

        notes = cast(list[str], d.pop("notes"))


        style = BrandStyle.from_dict(d.pop("style"))




        website = d.pop("website")

        _logo_dark = d.pop("logoDark", UNSET)
        logo_dark: BrandLogo | Unset
        if isinstance(_logo_dark,  Unset):
            logo_dark = UNSET
        else:
            logo_dark = BrandLogo.from_dict(_logo_dark)




        _logo_light = d.pop("logoLight", UNSET)
        logo_light: BrandLogo | Unset
        if isinstance(_logo_light,  Unset):
            logo_light = UNSET
        else:
            logo_light = BrandLogo.from_dict(_logo_light)




        brand_kit_import = cls(
            colors=colors,
            fonts=fonts,
            found_at=found_at,
            logo_alt=logo_alt,
            notes=notes,
            style=style,
            website=website,
            logo_dark=logo_dark,
            logo_light=logo_light,
        )

        return brand_kit_import

