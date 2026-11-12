from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.website_post_input_publish_mode import WebsitePostInputPublishMode
from ..models.website_post_input_visibility import WebsitePostInputVisibility
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.website_post_seo import WebsitePostSEO





T = TypeVar("T", bound="WebsitePostInput")



@_attrs_define
class WebsitePostInput:
    """ 
        Attributes:
            cover_alt (str):
            cover_url (str):
            excerpt (str):
            featured (bool):
            never_free (bool):
            publish_mode (WebsitePostInputPublishMode):
            seo (WebsitePostSEO):
            slug (str):
            tags (list[str]):
            tiers (list[str]): A paid post's tiers; empty for every paid tier
            title (str):
            visibility (WebsitePostInputVisibility):
            web_on (bool):
            byline_name (str | Unset):
            byline_user_id (str | Unset):
            publish_at (datetime.datetime | Unset):
            publish_tz (str | Unset):
            use_preheader (bool | Unset):
            use_subject (bool | Unset):
     """

    cover_alt: str
    cover_url: str
    excerpt: str
    featured: bool
    never_free: bool
    publish_mode: WebsitePostInputPublishMode
    seo: WebsitePostSEO
    slug: str
    tags: list[str]
    tiers: list[str]
    title: str
    visibility: WebsitePostInputVisibility
    web_on: bool
    byline_name: str | Unset = UNSET
    byline_user_id: str | Unset = UNSET
    publish_at: datetime.datetime | Unset = UNSET
    publish_tz: str | Unset = UNSET
    use_preheader: bool | Unset = UNSET
    use_subject: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.website_post_seo import WebsitePostSEO # noqa: PLC0415
        cover_alt = self.cover_alt

        cover_url = self.cover_url

        excerpt = self.excerpt

        featured = self.featured

        never_free = self.never_free

        publish_mode = self.publish_mode.value

        seo = self.seo.to_dict()

        slug = self.slug

        tags = self.tags



        tiers = self.tiers



        title = self.title

        visibility = self.visibility.value

        web_on = self.web_on

        byline_name = self.byline_name

        byline_user_id = self.byline_user_id

        publish_at: str | Unset = UNSET
        if not isinstance(self.publish_at, Unset):
            publish_at = self.publish_at.isoformat()

        publish_tz = self.publish_tz

        use_preheader = self.use_preheader

        use_subject = self.use_subject


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "coverAlt": cover_alt,
            "coverUrl": cover_url,
            "excerpt": excerpt,
            "featured": featured,
            "neverFree": never_free,
            "publishMode": publish_mode,
            "seo": seo,
            "slug": slug,
            "tags": tags,
            "tiers": tiers,
            "title": title,
            "visibility": visibility,
            "webOn": web_on,
        })
        if byline_name is not UNSET:
            field_dict["bylineName"] = byline_name
        if byline_user_id is not UNSET:
            field_dict["bylineUserId"] = byline_user_id
        if publish_at is not UNSET:
            field_dict["publishAt"] = publish_at
        if publish_tz is not UNSET:
            field_dict["publishTz"] = publish_tz
        if use_preheader is not UNSET:
            field_dict["usePreheader"] = use_preheader
        if use_subject is not UNSET:
            field_dict["useSubject"] = use_subject

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.website_post_seo import WebsitePostSEO # noqa: PLC0415
        d = dict(src_dict)
        cover_alt = d.pop("coverAlt")

        cover_url = d.pop("coverUrl")

        excerpt = d.pop("excerpt")

        featured = d.pop("featured")

        never_free = d.pop("neverFree")

        publish_mode = WebsitePostInputPublishMode(d.pop("publishMode"))




        seo = WebsitePostSEO.from_dict(d.pop("seo"))




        slug = d.pop("slug")

        tags = cast(list[str], d.pop("tags"))


        tiers = cast(list[str], d.pop("tiers"))


        title = d.pop("title")

        visibility = WebsitePostInputVisibility(d.pop("visibility"))




        web_on = d.pop("webOn")

        byline_name = d.pop("bylineName", UNSET)

        byline_user_id = d.pop("bylineUserId", UNSET)

        _publish_at = d.pop("publishAt", UNSET)
        publish_at: datetime.datetime | Unset
        if isinstance(_publish_at,  Unset):
            publish_at = UNSET
        else:
            publish_at = datetime.datetime.fromisoformat(_publish_at)




        publish_tz = d.pop("publishTz", UNSET)

        use_preheader = d.pop("usePreheader", UNSET)

        use_subject = d.pop("useSubject", UNSET)

        website_post_input = cls(
            cover_alt=cover_alt,
            cover_url=cover_url,
            excerpt=excerpt,
            featured=featured,
            never_free=never_free,
            publish_mode=publish_mode,
            seo=seo,
            slug=slug,
            tags=tags,
            tiers=tiers,
            title=title,
            visibility=visibility,
            web_on=web_on,
            byline_name=byline_name,
            byline_user_id=byline_user_id,
            publish_at=publish_at,
            publish_tz=publish_tz,
            use_preheader=use_preheader,
            use_subject=use_subject,
        )

        return website_post_input

