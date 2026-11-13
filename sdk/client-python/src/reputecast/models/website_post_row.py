from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.website_post_row_kind import WebsitePostRowKind
from ..models.website_post_row_publish_mode import WebsitePostRowPublishMode
from ..models.website_post_row_status import WebsitePostRowStatus
from ..models.website_post_row_visibility import WebsitePostRowVisibility
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.website_byline import WebsiteByline
  from ..models.website_post_approval import WebsitePostApproval
  from ..models.website_post_campaign import WebsitePostCampaign
  from ..models.website_tag_ref import WebsiteTagRef
  from ..models.website_tier_ref import WebsiteTierRef





T = TypeVar("T", bound="WebsitePostRow")



@_attrs_define
class WebsitePostRow:
    """ 
        Attributes:
            byline (WebsiteByline):
            comments (int):
            comments_pending (int):
            created_by (str):
            excerpt (str):
            featured (bool):
            has_marker (bool):
            id (str):
            kind (WebsitePostRowKind):
            marker_after (int):
            never_free (bool):
            publish_mode (WebsitePostRowPublishMode):
            signups (int):
            slug (str):
            status (WebsitePostRowStatus):
            tags (list[WebsiteTagRef]):
            tiers (list[WebsiteTierRef]):
            title (str):
            updated_at (datetime.datetime):
            upgrades (int):
            url (str):
            views (int):
            views_all_time (int):
            visibility (WebsitePostRowVisibility):
            approval (WebsitePostApproval | Unset):
            campaign (WebsitePostCampaign | Unset):
            publish_at (datetime.datetime | Unset):
            publish_tz (str | Unset):
            published_at (datetime.datetime | Unset):
            redirect_to (str | Unset):
            unpublish_reason (str | Unset):
            unpublished_at (datetime.datetime | Unset):
            unpublished_by (str | Unset):
     """

    byline: WebsiteByline
    comments: int
    comments_pending: int
    created_by: str
    excerpt: str
    featured: bool
    has_marker: bool
    id: str
    kind: WebsitePostRowKind
    marker_after: int
    never_free: bool
    publish_mode: WebsitePostRowPublishMode
    signups: int
    slug: str
    status: WebsitePostRowStatus
    tags: list[WebsiteTagRef]
    tiers: list[WebsiteTierRef]
    title: str
    updated_at: datetime.datetime
    upgrades: int
    url: str
    views: int
    views_all_time: int
    visibility: WebsitePostRowVisibility
    approval: WebsitePostApproval | Unset = UNSET
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
        from ..models.website_tag_ref import WebsiteTagRef # noqa: PLC0415
        from ..models.website_tier_ref import WebsiteTierRef # noqa: PLC0415
        byline = self.byline.to_dict()

        comments = self.comments

        comments_pending = self.comments_pending

        created_by = self.created_by

        excerpt = self.excerpt

        featured = self.featured

        has_marker = self.has_marker

        id = self.id

        kind = self.kind.value

        marker_after = self.marker_after

        never_free = self.never_free

        publish_mode = self.publish_mode.value

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

        updated_at = self.updated_at.isoformat()

        upgrades = self.upgrades

        url = self.url

        views = self.views

        views_all_time = self.views_all_time

        visibility = self.visibility.value

        approval: dict[str, Any] | Unset = UNSET
        if not isinstance(self.approval, Unset):
            approval = self.approval.to_dict()

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
            "byline": byline,
            "comments": comments,
            "commentsPending": comments_pending,
            "createdBy": created_by,
            "excerpt": excerpt,
            "featured": featured,
            "hasMarker": has_marker,
            "id": id,
            "kind": kind,
            "markerAfter": marker_after,
            "neverFree": never_free,
            "publishMode": publish_mode,
            "signups": signups,
            "slug": slug,
            "status": status,
            "tags": tags,
            "tiers": tiers,
            "title": title,
            "updatedAt": updated_at,
            "upgrades": upgrades,
            "url": url,
            "views": views,
            "viewsAllTime": views_all_time,
            "visibility": visibility,
        })
        if approval is not UNSET:
            field_dict["approval"] = approval
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
        from ..models.website_tag_ref import WebsiteTagRef # noqa: PLC0415
        from ..models.website_tier_ref import WebsiteTierRef # noqa: PLC0415
        d = dict(src_dict)
        byline = WebsiteByline.from_dict(d.pop("byline"))




        comments = d.pop("comments")

        comments_pending = d.pop("commentsPending")

        created_by = d.pop("createdBy")

        excerpt = d.pop("excerpt")

        featured = d.pop("featured")

        has_marker = d.pop("hasMarker")

        id = d.pop("id")

        kind = WebsitePostRowKind(d.pop("kind"))




        marker_after = d.pop("markerAfter")

        never_free = d.pop("neverFree")

        publish_mode = WebsitePostRowPublishMode(d.pop("publishMode"))




        signups = d.pop("signups")

        slug = d.pop("slug")

        status = WebsitePostRowStatus(d.pop("status"))




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

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))




        upgrades = d.pop("upgrades")

        url = d.pop("url")

        views = d.pop("views")

        views_all_time = d.pop("viewsAllTime")

        visibility = WebsitePostRowVisibility(d.pop("visibility"))




        _approval = d.pop("approval", UNSET)
        approval: WebsitePostApproval | Unset
        if isinstance(_approval,  Unset):
            approval = UNSET
        else:
            approval = WebsitePostApproval.from_dict(_approval)




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

        website_post_row = cls(
            byline=byline,
            comments=comments,
            comments_pending=comments_pending,
            created_by=created_by,
            excerpt=excerpt,
            featured=featured,
            has_marker=has_marker,
            id=id,
            kind=kind,
            marker_after=marker_after,
            never_free=never_free,
            publish_mode=publish_mode,
            signups=signups,
            slug=slug,
            status=status,
            tags=tags,
            tiers=tiers,
            title=title,
            updated_at=updated_at,
            upgrades=upgrades,
            url=url,
            views=views,
            views_all_time=views_all_time,
            visibility=visibility,
            approval=approval,
            campaign=campaign,
            publish_at=publish_at,
            publish_tz=publish_tz,
            published_at=published_at,
            redirect_to=redirect_to,
            unpublish_reason=unpublish_reason,
            unpublished_at=unpublished_at,
            unpublished_by=unpublished_by,
        )

        return website_post_row

