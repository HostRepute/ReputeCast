from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.brand_button_style import BrandButtonStyle
  from ..models.brand_colors import BrandColors
  from ..models.brand_font import BrandFont





T = TypeVar("T", bound="BrandStyle")



@_attrs_define
class BrandStyle:
    """ 
        Attributes:
            body (BrandFont):
            button (BrandButtonStyle):
            colors (BrandColors):
            heading (BrandFont):
     """

    body: BrandFont
    button: BrandButtonStyle
    colors: BrandColors
    heading: BrandFont





    def to_dict(self) -> dict[str, Any]:
        from ..models.brand_button_style import BrandButtonStyle # noqa: PLC0415
        from ..models.brand_colors import BrandColors # noqa: PLC0415
        from ..models.brand_font import BrandFont # noqa: PLC0415
        body = self.body.to_dict()

        button = self.button.to_dict()

        colors = self.colors.to_dict()

        heading = self.heading.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "body": body,
            "button": button,
            "colors": colors,
            "heading": heading,
        })

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




        brand_style = cls(
            body=body,
            button=button,
            colors=colors,
            heading=heading,
        )

        return brand_style

