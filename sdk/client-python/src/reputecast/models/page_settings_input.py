from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.page_settings_input_visibility import PageSettingsInputVisibility
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="PageSettingsInput")



@_attrs_define
class PageSettingsInput:
    """ 
        Attributes:
            hide_header (bool):
            name (str):
            nav_label (str):
            seo_description (str):
            seo_title (str):
            show_in_nav (bool):
            slug (str):
            social_image (str):
            visibility (PageSettingsInputVisibility):
            tiers (list[str] | Unset):
     """

    hide_header: bool
    name: str
    nav_label: str
    seo_description: str
    seo_title: str
    show_in_nav: bool
    slug: str
    social_image: str
    visibility: PageSettingsInputVisibility
    tiers: list[str] | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        hide_header = self.hide_header

        name = self.name

        nav_label = self.nav_label

        seo_description = self.seo_description

        seo_title = self.seo_title

        show_in_nav = self.show_in_nav

        slug = self.slug

        social_image = self.social_image

        visibility = self.visibility.value

        tiers: list[str] | Unset = UNSET
        if not isinstance(self.tiers, Unset):
            tiers = self.tiers




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "hideHeader": hide_header,
            "name": name,
            "navLabel": nav_label,
            "seoDescription": seo_description,
            "seoTitle": seo_title,
            "showInNav": show_in_nav,
            "slug": slug,
            "socialImage": social_image,
            "visibility": visibility,
        })
        if tiers is not UNSET:
            field_dict["tiers"] = tiers

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        hide_header = d.pop("hideHeader")

        name = d.pop("name")

        nav_label = d.pop("navLabel")

        seo_description = d.pop("seoDescription")

        seo_title = d.pop("seoTitle")

        show_in_nav = d.pop("showInNav")

        slug = d.pop("slug")

        social_image = d.pop("socialImage")

        visibility = PageSettingsInputVisibility(d.pop("visibility"))




        tiers = cast(list[str], d.pop("tiers", UNSET))


        page_settings_input = cls(
            hide_header=hide_header,
            name=name,
            nav_label=nav_label,
            seo_description=seo_description,
            seo_title=seo_title,
            show_in_nav=show_in_nav,
            slug=slug,
            social_image=social_image,
            visibility=visibility,
            tiers=tiers,
        )

        return page_settings_input

