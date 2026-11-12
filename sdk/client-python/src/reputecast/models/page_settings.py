from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.page_settings_visibility import PageSettingsVisibility
from typing import cast






T = TypeVar("T", bound="PageSettings")



@_attrs_define
class PageSettings:
    """ 
        Attributes:
            hide_header (bool): Focused landing pages: the site header shows the logo only
            nav_label (str):
            seo_description (str):
            seo_title (str):
            show_in_nav (bool):
            social_image (str):
            tiers (list[str]): Paid pages: the tiers that may read it (empty: every paid tier)
            visibility (PageSettingsVisibility):
     """

    hide_header: bool
    nav_label: str
    seo_description: str
    seo_title: str
    show_in_nav: bool
    social_image: str
    tiers: list[str]
    visibility: PageSettingsVisibility





    def to_dict(self) -> dict[str, Any]:
        hide_header = self.hide_header

        nav_label = self.nav_label

        seo_description = self.seo_description

        seo_title = self.seo_title

        show_in_nav = self.show_in_nav

        social_image = self.social_image

        tiers = self.tiers



        visibility = self.visibility.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "hideHeader": hide_header,
            "navLabel": nav_label,
            "seoDescription": seo_description,
            "seoTitle": seo_title,
            "showInNav": show_in_nav,
            "socialImage": social_image,
            "tiers": tiers,
            "visibility": visibility,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        hide_header = d.pop("hideHeader")

        nav_label = d.pop("navLabel")

        seo_description = d.pop("seoDescription")

        seo_title = d.pop("seoTitle")

        show_in_nav = d.pop("showInNav")

        social_image = d.pop("socialImage")

        tiers = cast(list[str], d.pop("tiers"))


        visibility = PageSettingsVisibility(d.pop("visibility"))




        page_settings = cls(
            hide_header=hide_header,
            nav_label=nav_label,
            seo_description=seo_description,
            seo_title=seo_title,
            show_in_nav=show_in_nav,
            social_image=social_image,
            tiers=tiers,
            visibility=visibility,
        )

        return page_settings

