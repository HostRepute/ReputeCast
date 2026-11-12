from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.website_home_nav_layout import WebsiteHomeNavLayout
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.website_about import WebsiteAbout
  from ..models.website_featured_post import WebsiteFeaturedPost
  from ..models.website_feed_ur_ls import WebsiteFeedURLs
  from ..models.website_footer import WebsiteFooter
  from ..models.website_home_section import WebsiteHomeSection
  from ..models.website_identity import WebsiteIdentity
  from ..models.website_nav_item import WebsiteNavItem
  from ..models.website_page_option import WebsitePageOption
  from ..models.website_tag import WebsiteTag





T = TypeVar("T", bound="WebsiteHomeNav")



@_attrs_define
class WebsiteHomeNav:
    """ 
        Attributes:
            about (WebsiteAbout):
            featured (list[WebsiteFeaturedPost]):
            footer (WebsiteFooter):
            header (list[WebsiteNavItem]):
            identity (WebsiteIdentity):
            index (bool):
            initial (str):
            layout (WebsiteHomeNavLayout):
            logo (str):
            pages (list[WebsitePageOption]):
            powered_by (bool):
            privacy_url (str):
            sections (list[WebsiteHomeSection]):
            share_buttons (bool):
            tags (list[WebsiteTag]):
            updated_at (datetime.datetime):
            urls (WebsiteFeedURLs):
            updated_by (str | Unset):
     """

    about: WebsiteAbout
    featured: list[WebsiteFeaturedPost]
    footer: WebsiteFooter
    header: list[WebsiteNavItem]
    identity: WebsiteIdentity
    index: bool
    initial: str
    layout: WebsiteHomeNavLayout
    logo: str
    pages: list[WebsitePageOption]
    powered_by: bool
    privacy_url: str
    sections: list[WebsiteHomeSection]
    share_buttons: bool
    tags: list[WebsiteTag]
    updated_at: datetime.datetime
    urls: WebsiteFeedURLs
    updated_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.website_about import WebsiteAbout # noqa: PLC0415
        from ..models.website_featured_post import WebsiteFeaturedPost # noqa: PLC0415
        from ..models.website_feed_ur_ls import WebsiteFeedURLs # noqa: PLC0415
        from ..models.website_footer import WebsiteFooter # noqa: PLC0415
        from ..models.website_home_section import WebsiteHomeSection # noqa: PLC0415
        from ..models.website_identity import WebsiteIdentity # noqa: PLC0415
        from ..models.website_nav_item import WebsiteNavItem # noqa: PLC0415
        from ..models.website_page_option import WebsitePageOption # noqa: PLC0415
        from ..models.website_tag import WebsiteTag # noqa: PLC0415
        about = self.about.to_dict()

        featured = []
        for featured_item_data in self.featured:
            featured_item = featured_item_data.to_dict()
            featured.append(featured_item)



        footer = self.footer.to_dict()

        header = []
        for header_item_data in self.header:
            header_item = header_item_data.to_dict()
            header.append(header_item)



        identity = self.identity.to_dict()

        index = self.index

        initial = self.initial

        layout = self.layout.value

        logo = self.logo

        pages = []
        for pages_item_data in self.pages:
            pages_item = pages_item_data.to_dict()
            pages.append(pages_item)



        powered_by = self.powered_by

        privacy_url = self.privacy_url

        sections = []
        for sections_item_data in self.sections:
            sections_item = sections_item_data.to_dict()
            sections.append(sections_item)



        share_buttons = self.share_buttons

        tags = []
        for tags_item_data in self.tags:
            tags_item = tags_item_data.to_dict()
            tags.append(tags_item)



        updated_at = self.updated_at.isoformat()

        urls = self.urls.to_dict()

        updated_by = self.updated_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "about": about,
            "featured": featured,
            "footer": footer,
            "header": header,
            "identity": identity,
            "index": index,
            "initial": initial,
            "layout": layout,
            "logo": logo,
            "pages": pages,
            "poweredBy": powered_by,
            "privacyUrl": privacy_url,
            "sections": sections,
            "shareButtons": share_buttons,
            "tags": tags,
            "updatedAt": updated_at,
            "urls": urls,
        })
        if updated_by is not UNSET:
            field_dict["updatedBy"] = updated_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.website_about import WebsiteAbout # noqa: PLC0415
        from ..models.website_featured_post import WebsiteFeaturedPost # noqa: PLC0415
        from ..models.website_feed_ur_ls import WebsiteFeedURLs # noqa: PLC0415
        from ..models.website_footer import WebsiteFooter # noqa: PLC0415
        from ..models.website_home_section import WebsiteHomeSection # noqa: PLC0415
        from ..models.website_identity import WebsiteIdentity # noqa: PLC0415
        from ..models.website_nav_item import WebsiteNavItem # noqa: PLC0415
        from ..models.website_page_option import WebsitePageOption # noqa: PLC0415
        from ..models.website_tag import WebsiteTag # noqa: PLC0415
        d = dict(src_dict)
        about = WebsiteAbout.from_dict(d.pop("about"))




        featured = []
        _featured = d.pop("featured")
        for featured_item_data in (_featured):
            featured_item = WebsiteFeaturedPost.from_dict(featured_item_data)



            featured.append(featured_item)


        footer = WebsiteFooter.from_dict(d.pop("footer"))




        header = []
        _header = d.pop("header")
        for header_item_data in (_header):
            header_item = WebsiteNavItem.from_dict(header_item_data)



            header.append(header_item)


        identity = WebsiteIdentity.from_dict(d.pop("identity"))




        index = d.pop("index")

        initial = d.pop("initial")

        layout = WebsiteHomeNavLayout(d.pop("layout"))




        logo = d.pop("logo")

        pages = []
        _pages = d.pop("pages")
        for pages_item_data in (_pages):
            pages_item = WebsitePageOption.from_dict(pages_item_data)



            pages.append(pages_item)


        powered_by = d.pop("poweredBy")

        privacy_url = d.pop("privacyUrl")

        sections = []
        _sections = d.pop("sections")
        for sections_item_data in (_sections):
            sections_item = WebsiteHomeSection.from_dict(sections_item_data)



            sections.append(sections_item)


        share_buttons = d.pop("shareButtons")

        tags = []
        _tags = d.pop("tags")
        for tags_item_data in (_tags):
            tags_item = WebsiteTag.from_dict(tags_item_data)



            tags.append(tags_item)


        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))




        urls = WebsiteFeedURLs.from_dict(d.pop("urls"))




        updated_by = d.pop("updatedBy", UNSET)

        website_home_nav = cls(
            about=about,
            featured=featured,
            footer=footer,
            header=header,
            identity=identity,
            index=index,
            initial=initial,
            layout=layout,
            logo=logo,
            pages=pages,
            powered_by=powered_by,
            privacy_url=privacy_url,
            sections=sections,
            share_buttons=share_buttons,
            tags=tags,
            updated_at=updated_at,
            urls=urls,
            updated_by=updated_by,
        )

        return website_home_nav

