from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.page_detail_status import PageDetailStatus
from ..models.page_detail_template import PageDetailTemplate
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.page_check import PageCheck
  from ..models.page_counts import PageCounts
  from ..models.page_draft import PageDraft
  from ..models.page_live import PageLive
  from ..models.page_numbers import PageNumbers
  from ..models.page_publish_state import PagePublishState
  from ..models.page_settings import PageSettings





T = TypeVar("T", bound="PageDetail")



@_attrs_define
class PageDetail:
    """ 
        Attributes:
            changes (bool):
            checks (list[PageCheck]):
            counts (PageCounts):
            created_at (datetime.datetime):
            draft (PageDraft):
            host (str):
            id (str):
            name (str):
            nav_full (bool):
            numbers (PageNumbers):
            path (str):
            publish (PagePublishState):
            settings (PageSettings):
            slug (str):
            status (PageDetailStatus):
            template (PageDetailTemplate):
            url (str):
            versions (int):
            website_on (bool):
            live (PageLive | Unset):
            publish_at (datetime.datetime | Unset):
            publish_tz (str | Unset):
            redirect_to (str | Unset):
            unpublish_reason (str | Unset):
            unpublished_at (datetime.datetime | Unset):
     """

    changes: bool
    checks: list[PageCheck]
    counts: PageCounts
    created_at: datetime.datetime
    draft: PageDraft
    host: str
    id: str
    name: str
    nav_full: bool
    numbers: PageNumbers
    path: str
    publish: PagePublishState
    settings: PageSettings
    slug: str
    status: PageDetailStatus
    template: PageDetailTemplate
    url: str
    versions: int
    website_on: bool
    live: PageLive | Unset = UNSET
    publish_at: datetime.datetime | Unset = UNSET
    publish_tz: str | Unset = UNSET
    redirect_to: str | Unset = UNSET
    unpublish_reason: str | Unset = UNSET
    unpublished_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.page_check import PageCheck # noqa: PLC0415
        from ..models.page_counts import PageCounts # noqa: PLC0415
        from ..models.page_draft import PageDraft # noqa: PLC0415
        from ..models.page_live import PageLive # noqa: PLC0415
        from ..models.page_numbers import PageNumbers # noqa: PLC0415
        from ..models.page_publish_state import PagePublishState # noqa: PLC0415
        from ..models.page_settings import PageSettings # noqa: PLC0415
        changes = self.changes

        checks = []
        for checks_item_data in self.checks:
            checks_item = checks_item_data.to_dict()
            checks.append(checks_item)



        counts = self.counts.to_dict()

        created_at = self.created_at.isoformat()

        draft = self.draft.to_dict()

        host = self.host

        id = self.id

        name = self.name

        nav_full = self.nav_full

        numbers = self.numbers.to_dict()

        path = self.path

        publish = self.publish.to_dict()

        settings = self.settings.to_dict()

        slug = self.slug

        status = self.status.value

        template = self.template.value

        url = self.url

        versions = self.versions

        website_on = self.website_on

        live: dict[str, Any] | Unset = UNSET
        if not isinstance(self.live, Unset):
            live = self.live.to_dict()

        publish_at: str | Unset = UNSET
        if not isinstance(self.publish_at, Unset):
            publish_at = self.publish_at.isoformat()

        publish_tz = self.publish_tz

        redirect_to = self.redirect_to

        unpublish_reason = self.unpublish_reason

        unpublished_at: str | Unset = UNSET
        if not isinstance(self.unpublished_at, Unset):
            unpublished_at = self.unpublished_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "changes": changes,
            "checks": checks,
            "counts": counts,
            "createdAt": created_at,
            "draft": draft,
            "host": host,
            "id": id,
            "name": name,
            "navFull": nav_full,
            "numbers": numbers,
            "path": path,
            "publish": publish,
            "settings": settings,
            "slug": slug,
            "status": status,
            "template": template,
            "url": url,
            "versions": versions,
            "websiteOn": website_on,
        })
        if live is not UNSET:
            field_dict["live"] = live
        if publish_at is not UNSET:
            field_dict["publishAt"] = publish_at
        if publish_tz is not UNSET:
            field_dict["publishTz"] = publish_tz
        if redirect_to is not UNSET:
            field_dict["redirectTo"] = redirect_to
        if unpublish_reason is not UNSET:
            field_dict["unpublishReason"] = unpublish_reason
        if unpublished_at is not UNSET:
            field_dict["unpublishedAt"] = unpublished_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.page_check import PageCheck # noqa: PLC0415
        from ..models.page_counts import PageCounts # noqa: PLC0415
        from ..models.page_draft import PageDraft # noqa: PLC0415
        from ..models.page_live import PageLive # noqa: PLC0415
        from ..models.page_numbers import PageNumbers # noqa: PLC0415
        from ..models.page_publish_state import PagePublishState # noqa: PLC0415
        from ..models.page_settings import PageSettings # noqa: PLC0415
        d = dict(src_dict)
        changes = d.pop("changes")

        checks = []
        _checks = d.pop("checks")
        for checks_item_data in (_checks):
            checks_item = PageCheck.from_dict(checks_item_data)



            checks.append(checks_item)


        counts = PageCounts.from_dict(d.pop("counts"))




        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        draft = PageDraft.from_dict(d.pop("draft"))




        host = d.pop("host")

        id = d.pop("id")

        name = d.pop("name")

        nav_full = d.pop("navFull")

        numbers = PageNumbers.from_dict(d.pop("numbers"))




        path = d.pop("path")

        publish = PagePublishState.from_dict(d.pop("publish"))




        settings = PageSettings.from_dict(d.pop("settings"))




        slug = d.pop("slug")

        status = PageDetailStatus(d.pop("status"))




        template = PageDetailTemplate(d.pop("template"))




        url = d.pop("url")

        versions = d.pop("versions")

        website_on = d.pop("websiteOn")

        _live = d.pop("live", UNSET)
        live: PageLive | Unset
        if isinstance(_live,  Unset):
            live = UNSET
        else:
            live = PageLive.from_dict(_live)




        _publish_at = d.pop("publishAt", UNSET)
        publish_at: datetime.datetime | Unset
        if isinstance(_publish_at,  Unset):
            publish_at = UNSET
        else:
            publish_at = datetime.datetime.fromisoformat(_publish_at)




        publish_tz = d.pop("publishTz", UNSET)

        redirect_to = d.pop("redirectTo", UNSET)

        unpublish_reason = d.pop("unpublishReason", UNSET)

        _unpublished_at = d.pop("unpublishedAt", UNSET)
        unpublished_at: datetime.datetime | Unset
        if isinstance(_unpublished_at,  Unset):
            unpublished_at = UNSET
        else:
            unpublished_at = datetime.datetime.fromisoformat(_unpublished_at)




        page_detail = cls(
            changes=changes,
            checks=checks,
            counts=counts,
            created_at=created_at,
            draft=draft,
            host=host,
            id=id,
            name=name,
            nav_full=nav_full,
            numbers=numbers,
            path=path,
            publish=publish,
            settings=settings,
            slug=slug,
            status=status,
            template=template,
            url=url,
            versions=versions,
            website_on=website_on,
            live=live,
            publish_at=publish_at,
            publish_tz=publish_tz,
            redirect_to=redirect_to,
            unpublish_reason=unpublish_reason,
            unpublished_at=unpublished_at,
        )

        return page_detail

