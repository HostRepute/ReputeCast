from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.website_home_nav_input_layout import WebsiteHomeNavInputLayout
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.website_about import WebsiteAbout
  from ..models.website_footer import WebsiteFooter
  from ..models.website_home_section import WebsiteHomeSection
  from ..models.website_identity import WebsiteIdentity
  from ..models.website_nav_link import WebsiteNavLink





T = TypeVar("T", bound="WebsiteHomeNavInput")



@_attrs_define
class WebsiteHomeNavInput:
    """ 
        Attributes:
            about (WebsiteAbout):
            featured (list[str]):
            footer (WebsiteFooter):
            header (list[WebsiteNavLink]):
            identity (WebsiteIdentity):
            index (bool):
            layout (WebsiteHomeNavInputLayout):
            powered_by (bool):
            privacy_url (str):
            sections (list[WebsiteHomeSection]):
            share_buttons (bool | Unset):
     """

    about: WebsiteAbout
    featured: list[str]
    footer: WebsiteFooter
    header: list[WebsiteNavLink]
    identity: WebsiteIdentity
    index: bool
    layout: WebsiteHomeNavInputLayout
    powered_by: bool
    privacy_url: str
    sections: list[WebsiteHomeSection]
    share_buttons: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.website_about import WebsiteAbout # noqa: PLC0415
        from ..models.website_footer import WebsiteFooter # noqa: PLC0415
        from ..models.website_home_section import WebsiteHomeSection # noqa: PLC0415
        from ..models.website_identity import WebsiteIdentity # noqa: PLC0415
        from ..models.website_nav_link import WebsiteNavLink # noqa: PLC0415
        about = self.about.to_dict()

        featured = self.featured



        footer = self.footer.to_dict()

        header = []
        for header_item_data in self.header:
            header_item = header_item_data.to_dict()
            header.append(header_item)



        identity = self.identity.to_dict()

        index = self.index

        layout = self.layout.value

        powered_by = self.powered_by

        privacy_url = self.privacy_url

        sections = []
        for sections_item_data in self.sections:
            sections_item = sections_item_data.to_dict()
            sections.append(sections_item)



        share_buttons = self.share_buttons


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "about": about,
            "featured": featured,
            "footer": footer,
            "header": header,
            "identity": identity,
            "index": index,
            "layout": layout,
            "poweredBy": powered_by,
            "privacyUrl": privacy_url,
            "sections": sections,
        })
        if share_buttons is not UNSET:
            field_dict["shareButtons"] = share_buttons

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.website_about import WebsiteAbout # noqa: PLC0415
        from ..models.website_footer import WebsiteFooter # noqa: PLC0415
        from ..models.website_home_section import WebsiteHomeSection # noqa: PLC0415
        from ..models.website_identity import WebsiteIdentity # noqa: PLC0415
        from ..models.website_nav_link import WebsiteNavLink # noqa: PLC0415
        d = dict(src_dict)
        about = WebsiteAbout.from_dict(d.pop("about"))




        featured = cast(list[str], d.pop("featured"))


        footer = WebsiteFooter.from_dict(d.pop("footer"))




        header = []
        _header = d.pop("header")
        for header_item_data in (_header):
            header_item = WebsiteNavLink.from_dict(header_item_data)



            header.append(header_item)


        identity = WebsiteIdentity.from_dict(d.pop("identity"))




        index = d.pop("index")

        layout = WebsiteHomeNavInputLayout(d.pop("layout"))




        powered_by = d.pop("poweredBy")

        privacy_url = d.pop("privacyUrl")

        sections = []
        _sections = d.pop("sections")
        for sections_item_data in (_sections):
            sections_item = WebsiteHomeSection.from_dict(sections_item_data)



            sections.append(sections_item)


        share_buttons = d.pop("shareButtons", UNSET)

        website_home_nav_input = cls(
            about=about,
            featured=featured,
            footer=footer,
            header=header,
            identity=identity,
            index=index,
            layout=layout,
            powered_by=powered_by,
            privacy_url=privacy_url,
            sections=sections,
            share_buttons=share_buttons,
        )

        return website_home_nav_input

