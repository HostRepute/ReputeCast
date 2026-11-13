from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.website_post_kind import WebsitePostKind
from ..models.website_post_publish_mode import WebsitePostPublishMode
from ..models.website_post_status import WebsitePostStatus
from ..models.website_post_visibility import WebsitePostVisibility
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.website_byline import WebsiteByline
  from ..models.website_post_approval import WebsitePostApproval
  from ..models.website_post_campaign import WebsitePostCampaign
  from ..models.website_post_seo import WebsitePostSEO
  from ..models.website_publish_state import WebsitePublishState
  from ..models.website_tag_ref import WebsiteTagRef
  from ..models.website_tier_ref import WebsiteTierRef
  from ..models.website_web_check import WebsiteWebCheck





T = TypeVar("T", bound="WebsitePost")



@_attrs_define
class WebsitePost:
    """ 
        Attributes:
            base (str):
            byline (WebsiteByline):
            bylines (list[WebsiteByline]):
            checks (list[WebsiteWebCheck]):
            comments (int):
            comments_off (bool):
            comments_pending (int):
            cover_alt (str):
            cover_url (str):
            created_by (str):
            excerpt (str):
            excerpt_edited (bool):
            featured (bool):
            has_marker (bool):
            id (str):
            kind (WebsitePostKind):
            marker_after (int):
            mjml (str):
            never_free (bool):
            publish (WebsitePublishState):
            publish_mode (WebsitePostPublishMode):
            published_version (int):
            reading_minutes (int):
            seo (WebsitePostSEO):
            signups (int):
            slug (str):
            status (WebsitePostStatus):
            tags (list[WebsiteTagRef]):
            tiers (list[WebsiteTierRef]):
            title (str):
            title_edited (bool):
            updated_at (datetime.datetime):
            upgrades (int):
            url (str):
            version (int):
            views (int):
            views_all_time (int):
            visibility (WebsitePostVisibility):
            web_on (bool):
            words (int):
            approval (WebsitePostApproval | Unset):
            autosaved_at (datetime.datetime | Unset):
            campaign (WebsitePostCampaign | Unset):
            publish_at (datetime.datetime | Unset):
            publish_tz (str | Unset):
            published_at (datetime.datetime | Unset):
            redirect_to (str | Unset):
            unpublish_reason (str | Unset):
            unpublished_at (datetime.datetime | Unset):
            unpublished_by (str | Unset):
     """

    base: str
    byline: WebsiteByline
    bylines: list[WebsiteByline]
    checks: list[WebsiteWebCheck]
    comments: int
    comments_off: bool
    comments_pending: int
    cover_alt: str
    cover_url: str
    created_by: str
    excerpt: str
    excerpt_edited: bool
    featured: bool
    has_marker: bool
    id: str
    kind: WebsitePostKind
    marker_after: int
    mjml: str
    never_free: bool
    publish: WebsitePublishState
    publish_mode: WebsitePostPublishMode
    published_version: int
    reading_minutes: int
    seo: WebsitePostSEO
    signups: int
    slug: str
    status: WebsitePostStatus
    tags: list[WebsiteTagRef]
    tiers: list[WebsiteTierRef]
    title: str
    title_edited: bool
    updated_at: datetime.datetime
    upgrades: int
    url: str
    version: int
    views: int
    views_all_time: int
    visibility: WebsitePostVisibility
    web_on: bool
    words: int
    approval: WebsitePostApproval | Unset = UNSET
    autosaved_at: datetime.datetime | Unset = UNSET
    campaign: WebsitePostCampaign | Unset = UNSET
    publish_at: datetime.datetime | Unset = UNSET
    publish_tz: str | Unset = UNSET
    published_at: datetime.datetime | Unset = UNSET
    redirect_to: str | Unset = UNSET
    unpublish_reason: str | Unset = UNSET
    unpublished_at: datetime.datetime | Unset = UNSET
    unpublished_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.website_byline import WebsiteByline # noqa: PLC0415
        from ..models.website_post_approval import WebsitePostApproval # noqa: PLC0415
        from ..models.website_post_campaign import WebsitePostCampaign # noqa: PLC0415
        from ..models.website_post_seo import WebsitePostSEO # noqa: PLC0415
        from ..models.website_publish_state import WebsitePublishState # noqa: PLC0415
        from ..models.website_tag_ref import WebsiteTagRef # noqa: PLC0415
        from ..models.website_tier_ref import WebsiteTierRef # noqa: PLC0415
        from ..models.website_web_check import WebsiteWebCheck # noqa: PLC0415
        base = self.base

        byline = self.byline.to_dict()

        bylines = []
        for bylines_item_data in self.bylines:
            bylines_item = bylines_item_data.to_dict()
            bylines.append(bylines_item)



        checks = []
        for checks_item_data in self.checks:
            checks_item = checks_item_data.to_dict()
            checks.append(checks_item)



        comments = self.comments

        comments_off = self.comments_off

        comments_pending = self.comments_pending

        cover_alt = self.cover_alt

        cover_url = self.cover_url

        created_by = self.created_by

        excerpt = self.excerpt

        excerpt_edited = self.excerpt_edited

        featured = self.featured

        has_marker = self.has_marker

        id = self.id

        kind = self.kind.value

        marker_after = self.marker_after

        mjml = self.mjml

        never_free = self.never_free

        publish = self.publish.to_dict()

        publish_mode = self.publish_mode.value

        published_version = self.published_version

        reading_minutes = self.reading_minutes

        seo = self.seo.to_dict()

        signups = self.signups

        slug = self.slug

        status = self.status.value

        tags = []
        for tags_item_data in self.tags:
            tags_item = tags_item_data.to_dict()
            tags.append(tags_item)



        tiers = []
        for tiers_item_data in self.tiers:
            tiers_item = tiers_item_data.to_dict()
            tiers.append(tiers_item)



        title = self.title

        title_edited = self.title_edited

        updated_at = self.updated_at.isoformat()

        upgrades = self.upgrades

        url = self.url

        version = self.version

        views = self.views

        views_all_time = self.views_all_time

        visibility = self.visibility.value

        web_on = self.web_on

        words = self.words

        approval: dict[str, Any] | Unset = UNSET
        if not isinstance(self.approval, Unset):
            approval = self.approval.to_dict()

        autosaved_at: str | Unset = UNSET
        if not isinstance(self.autosaved_at, Unset):
            autosaved_at = self.autosaved_at.isoformat()

        campaign: dict[str, Any] | Unset = UNSET
        if not isinstance(self.campaign, Unset):
            campaign = self.campaign.to_dict()

        publish_at: str | Unset = UNSET
        if not isinstance(self.publish_at, Unset):
            publish_at = self.publish_at.isoformat()

        publish_tz = self.publish_tz

        published_at: str | Unset = UNSET
        if not isinstance(self.published_at, Unset):
            published_at = self.published_at.isoformat()

        redirect_to = self.redirect_to

        unpublish_reason = self.unpublish_reason

        unpublished_at: str | Unset = UNSET
        if not isinstance(self.unpublished_at, Unset):
            unpublished_at = self.unpublished_at.isoformat()

        unpublished_by = self.unpublished_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "base": base,
            "byline": byline,
            "bylines": bylines,
            "checks": checks,
            "comments": comments,
            "commentsOff": comments_off,
            "commentsPending": comments_pending,
            "coverAlt": cover_alt,
            "coverUrl": cover_url,
            "createdBy": created_by,
            "excerpt": excerpt,
            "excerptEdited": excerpt_edited,
            "featured": featured,
            "hasMarker": has_marker,
            "id": id,
            "kind": kind,
            "markerAfter": marker_after,
            "mjml": mjml,
            "neverFree": never_free,
            "publish": publish,
            "publishMode": publish_mode,
            "publishedVersion": published_version,
            "readingMinutes": reading_minutes,
            "seo": seo,
            "signups": signups,
            "slug": slug,
            "status": status,
            "tags": tags,
            "tiers": tiers,
            "title": title,
            "titleEdited": title_edited,
            "updatedAt": updated_at,
            "upgrades": upgrades,
            "url": url,
            "version": version,
            "views": views,
            "viewsAllTime": views_all_time,
            "visibility": visibility,
            "webOn": web_on,
            "words": words,
        })
        if approval is not UNSET:
            field_dict["approval"] = approval
        if autosaved_at is not UNSET:
            field_dict["autosavedAt"] = autosaved_at
        if campaign is not UNSET:
            field_dict["campaign"] = campaign
        if publish_at is not UNSET:
            field_dict["publishAt"] = publish_at
        if publish_tz is not UNSET:
            field_dict["publishTz"] = publish_tz
        if published_at is not UNSET:
            field_dict["publishedAt"] = published_at
        if redirect_to is not UNSET:
            field_dict["redirectTo"] = redirect_to
        if unpublish_reason is not UNSET:
            field_dict["unpublishReason"] = unpublish_reason
        if unpublished_at is not UNSET:
            field_dict["unpublishedAt"] = unpublished_at
        if unpublished_by is not UNSET:
            field_dict["unpublishedBy"] = unpublished_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.website_byline import WebsiteByline # noqa: PLC0415
        from ..models.website_post_approval import WebsitePostApproval # noqa: PLC0415
        from ..models.website_post_campaign import WebsitePostCampaign # noqa: PLC0415
        from ..models.website_post_seo import WebsitePostSEO # noqa: PLC0415
        from ..models.website_publish_state import WebsitePublishState # noqa: PLC0415
        from ..models.website_tag_ref import WebsiteTagRef # noqa: PLC0415
        from ..models.website_tier_ref import WebsiteTierRef # noqa: PLC0415
        from ..models.website_web_check import WebsiteWebCheck # noqa: PLC0415
        d = dict(src_dict)
        base = d.pop("base")

        byline = WebsiteByline.from_dict(d.pop("byline"))




        bylines = []
        _bylines = d.pop("bylines")
        for bylines_item_data in (_bylines):
            bylines_item = WebsiteByline.from_dict(bylines_item_data)



            bylines.append(bylines_item)


        checks = []
        _checks = d.pop("checks")
        for checks_item_data in (_checks):
            checks_item = WebsiteWebCheck.from_dict(checks_item_data)



            checks.append(checks_item)


        comments = d.pop("comments")

        comments_off = d.pop("commentsOff")

        comments_pending = d.pop("commentsPending")

        cover_alt = d.pop("coverAlt")

        cover_url = d.pop("coverUrl")

        created_by = d.pop("createdBy")

        excerpt = d.pop("excerpt")

        excerpt_edited = d.pop("excerptEdited")

        featured = d.pop("featured")

        has_marker = d.pop("hasMarker")

        id = d.pop("id")

        kind = WebsitePostKind(d.pop("kind"))




        marker_after = d.pop("markerAfter")

        mjml = d.pop("mjml")

        never_free = d.pop("neverFree")

        publish = WebsitePublishState.from_dict(d.pop("publish"))




        publish_mode = WebsitePostPublishMode(d.pop("publishMode"))




        published_version = d.pop("publishedVersion")

        reading_minutes = d.pop("readingMinutes")

        seo = WebsitePostSEO.from_dict(d.pop("seo"))




        signups = d.pop("signups")

        slug = d.pop("slug")

        status = WebsitePostStatus(d.pop("status"))




        tags = []
        _tags = d.pop("tags")
        for tags_item_data in (_tags):
            tags_item = WebsiteTagRef.from_dict(tags_item_data)



            tags.append(tags_item)


        tiers = []
        _tiers = d.pop("tiers")
        for tiers_item_data in (_tiers):
            tiers_item = WebsiteTierRef.from_dict(tiers_item_data)



            tiers.append(tiers_item)


        title = d.pop("title")

        title_edited = d.pop("titleEdited")

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))




        upgrades = d.pop("upgrades")

        url = d.pop("url")

        version = d.pop("version")

        views = d.pop("views")

        views_all_time = d.pop("viewsAllTime")

        visibility = WebsitePostVisibility(d.pop("visibility"))




        web_on = d.pop("webOn")

        words = d.pop("words")

        _approval = d.pop("approval", UNSET)
        approval: WebsitePostApproval | Unset
        if isinstance(_approval,  Unset):
            approval = UNSET
        else:
            approval = WebsitePostApproval.from_dict(_approval)




        _autosaved_at = d.pop("autosavedAt", UNSET)
        autosaved_at: datetime.datetime | Unset
        if isinstance(_autosaved_at,  Unset):
            autosaved_at = UNSET
        else:
            autosaved_at = datetime.datetime.fromisoformat(_autosaved_at)




        _campaign = d.pop("campaign", UNSET)
        campaign: WebsitePostCampaign | Unset
        if isinstance(_campaign,  Unset):
            campaign = UNSET
        else:
            campaign = WebsitePostCampaign.from_dict(_campaign)




        _publish_at = d.pop("publishAt", UNSET)
        publish_at: datetime.datetime | Unset
        if isinstance(_publish_at,  Unset):
            publish_at = UNSET
        else:
            publish_at = datetime.datetime.fromisoformat(_publish_at)




        publish_tz = d.pop("publishTz", UNSET)

        _published_at = d.pop("publishedAt", UNSET)
        published_at: datetime.datetime | Unset
        if isinstance(_published_at,  Unset):
            published_at = UNSET
        else:
            published_at = datetime.datetime.fromisoformat(_published_at)




        redirect_to = d.pop("redirectTo", UNSET)

        unpublish_reason = d.pop("unpublishReason", UNSET)

        _unpublished_at = d.pop("unpublishedAt", UNSET)
        unpublished_at: datetime.datetime | Unset
        if isinstance(_unpublished_at,  Unset):
            unpublished_at = UNSET
        else:
            unpublished_at = datetime.datetime.fromisoformat(_unpublished_at)




        unpublished_by = d.pop("unpublishedBy", UNSET)

        website_post = cls(
            base=base,
            byline=byline,
            bylines=bylines,
            checks=checks,
            comments=comments,
            comments_off=comments_off,
            comments_pending=comments_pending,
            cover_alt=cover_alt,
            cover_url=cover_url,
            created_by=created_by,
            excerpt=excerpt,
            excerpt_edited=excerpt_edited,
            featured=featured,
            has_marker=has_marker,
            id=id,
            kind=kind,
            marker_after=marker_after,
            mjml=mjml,
            never_free=never_free,
            publish=publish,
            publish_mode=publish_mode,
            published_version=published_version,
            reading_minutes=reading_minutes,
            seo=seo,
            signups=signups,
            slug=slug,
            status=status,
            tags=tags,
            tiers=tiers,
            title=title,
            title_edited=title_edited,
            updated_at=updated_at,
            upgrades=upgrades,
            url=url,
            version=version,
            views=views,
            views_all_time=views_all_time,
            visibility=visibility,
            web_on=web_on,
            words=words,
            approval=approval,
            autosaved_at=autosaved_at,
            campaign=campaign,
            publish_at=publish_at,
            publish_tz=publish_tz,
            published_at=published_at,
            redirect_to=redirect_to,
            unpublish_reason=unpublish_reason,
            unpublished_at=unpublished_at,
            unpublished_by=unpublished_by,
        )

        return website_post

