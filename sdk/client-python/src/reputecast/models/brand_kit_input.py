from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.brand_button_style import BrandButtonStyle
  from ..models.brand_colors import BrandColors
  from ..models.brand_font import BrandFont





T = TypeVar("T", bound="BrandKitInput")



@_attrs_define
class BrandKitInput:
    """ 
        Attributes:
            body (BrandFont):
            button (BrandButtonStyle):
            colors (BrandColors):
            heading (BrandFont):
            logo_alt (str):
            logo_dark_id (str | Unset):
            logo_light_id (str | Unset): A media library image id; empty: no logo
            website (str | Unset):
     """

    body: BrandFont
    button: BrandButtonStyle
    colors: BrandColors
    heading: BrandFont
    logo_alt: str
    logo_dark_id: str | Unset = UNSET
    logo_light_id: str | Unset = UNSET
    website: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.brand_button_style import BrandButtonStyle # noqa: PLC0415
        from ..models.brand_colors import BrandColors # noqa: PLC0415
        from ..models.brand_font import BrandFont # noqa: PLC0415
        body = self.body.to_dict()

        button = self.button.to_dict()

        colors = self.colors.to_dict()

        heading = self.heading.to_dict()

        logo_alt = self.logo_alt

        logo_dark_id = self.logo_dark_id

        logo_light_id = self.logo_light_id

        website = self.website


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "body": body,
            "button": button,
            "colors": colors,
            "heading": heading,
            "logoAlt": logo_alt,
        })
        if logo_dark_id is not UNSET:
            field_dict["logoDarkId"] = logo_dark_id
        if logo_light_id is not UNSET:
            field_dict["logoLightId"] = logo_light_id
        if website is not UNSET:
            field_dict["website"] = website

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.brand_button_style import BrandButtonStyle # noqa: PLC0415
        from ..models.brand_colors import BrandColors # noqa: PLC0415
        from ..models.brand_font import BrandFont # noqa: PLC0415
        d = dict(src_dict)
        body = BrandFont.from_dict(d.pop("body"))




        button = BrandButtonStyle.from_dict(d.pop("button"))




        colors = BrandColors.from_dict(d.pop("colors"))




        heading = BrandFont.from_dict(d.pop("heading"))




        logo_alt = d.pop("logoAlt")

        logo_dark_id = d.pop("logoDarkId", UNSET)

        logo_light_id = d.pop("logoLightId", UNSET)

        website = d.pop("website", UNSET)

        brand_kit_input = cls(
            body=body,
            button=button,
            colors=colors,
            heading=heading,
            logo_alt=logo_alt,
            logo_dark_id=logo_dark_id,
            logo_light_id=logo_light_id,
            website=website,
        )

        return brand_kit_input

