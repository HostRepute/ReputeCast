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
  from ..models.brand_button_style import BrandButtonStyle
  from ..models.brand_colors import BrandColors
  from ..models.brand_font import BrandFont
  from ..models.brand_logo import BrandLogo
  from ..models.synced_block import SyncedBlock





T = TypeVar("T", bound="BrandKit")



@_attrs_define
class BrandKit:
    """ 
        Attributes:
            body (BrandFont):
            button (BrandButtonStyle):
            colors (BrandColors):
            heading (BrandFont):
            logo_alt (str):
            saved (bool):
            synced (list[SyncedBlock]):
            website (str):
            logo_dark (BrandLogo | Unset):
            logo_light (BrandLogo | Unset):
            updated_at (datetime.datetime | Unset):
            updated_by (str | Unset):
     """

    body: BrandFont
    button: BrandButtonStyle
    colors: BrandColors
    heading: BrandFont
    logo_alt: str
    saved: bool
    synced: list[SyncedBlock]
    website: str
    logo_dark: BrandLogo | Unset = UNSET
    logo_light: BrandLogo | Unset = UNSET
    updated_at: datetime.datetime | Unset = UNSET
    updated_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.brand_button_style import BrandButtonStyle # noqa: PLC0415
        from ..models.brand_colors import BrandColors # noqa: PLC0415
        from ..models.brand_font import BrandFont # noqa: PLC0415
        from ..models.brand_logo import BrandLogo # noqa: PLC0415
        from ..models.synced_block import SyncedBlock # noqa: PLC0415
        body = self.body.to_dict()

        button = self.button.to_dict()

        colors = self.colors.to_dict()

        heading = self.heading.to_dict()

        logo_alt = self.logo_alt

        saved = self.saved

        synced = []
        for synced_item_data in self.synced:
            synced_item = synced_item_data.to_dict()
            synced.append(synced_item)



        website = self.website

        logo_dark: dict[str, Any] | Unset = UNSET
        if not isinstance(self.logo_dark, Unset):
            logo_dark = self.logo_dark.to_dict()

        logo_light: dict[str, Any] | Unset = UNSET
        if not isinstance(self.logo_light, Unset):
            logo_light = self.logo_light.to_dict()

        updated_at: str | Unset = UNSET
        if not isinstance(self.updated_at, Unset):
            updated_at = self.updated_at.isoformat()

        updated_by = self.updated_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "body": body,
            "button": button,
            "colors": colors,
            "heading": heading,
            "logoAlt": logo_alt,
            "saved": saved,
            "synced": synced,
            "website": website,
        })
        if logo_dark is not UNSET:
            field_dict["logoDark"] = logo_dark
        if logo_light is not UNSET:
            field_dict["logoLight"] = logo_light
        if updated_at is not UNSET:
            field_dict["updatedAt"] = updated_at
        if updated_by is not UNSET:
            field_dict["updatedBy"] = updated_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.brand_button_style import BrandButtonStyle # noqa: PLC0415
        from ..models.brand_colors import BrandColors # noqa: PLC0415
        from ..models.brand_font import BrandFont # noqa: PLC0415
        from ..models.brand_logo import BrandLogo # noqa: PLC0415
        from ..models.synced_block import SyncedBlock # noqa: PLC0415
        d = dict(src_dict)
        body = BrandFont.from_dict(d.pop("body"))




        button = BrandButtonStyle.from_dict(d.pop("button"))




        colors = BrandColors.from_dict(d.pop("colors"))




        heading = BrandFont.from_dict(d.pop("heading"))




        logo_alt = d.pop("logoAlt")

        saved = d.pop("saved")

        synced = []
        _synced = d.pop("synced")
        for synced_item_data in (_synced):
            synced_item = SyncedBlock.from_dict(synced_item_data)



            synced.append(synced_item)


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




        _updated_at = d.pop("updatedAt", UNSET)
        updated_at: datetime.datetime | Unset
        if isinstance(_updated_at,  Unset):
            updated_at = UNSET
        else:
            updated_at = datetime.datetime.fromisoformat(_updated_at)




        updated_by = d.pop("updatedBy", UNSET)

        brand_kit = cls(
            body=body,
            button=button,
            colors=colors,
            heading=heading,
            logo_alt=logo_alt,
            saved=saved,
            synced=synced,
            website=website,
            logo_dark=logo_dark,
            logo_light=logo_light,
            updated_at=updated_at,
            updated_by=updated_by,
        )

        return brand_kit

