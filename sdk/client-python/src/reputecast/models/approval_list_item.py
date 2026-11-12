from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.approval_list_item_channel import ApprovalListItemChannel
from ..models.approval_list_item_schedule_mode import ApprovalListItemScheduleMode
from ..models.approval_list_item_status import ApprovalListItemStatus
from ..models.approval_list_item_subject_type import ApprovalListItemSubjectType
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.approval_page_snapshot import ApprovalPageSnapshot
  from ..models.approval_person import ApprovalPerson
  from ..models.approval_website_snapshot import ApprovalWebsiteSnapshot





T = TypeVar("T", bound="ApprovalListItem")



@_attrs_define
class ApprovalListItem:
    """ 
        Attributes:
            audience (int):
            author (ApprovalPerson):
            channel (ApprovalListItemChannel):
            id (str):
            name (str):
            requested_at (datetime.datetime):
            route (str):
            rules (str):
            schedule_mode (ApprovalListItemScheduleMode):
            status (ApprovalListItemStatus):
            subject_type (ApprovalListItemSubjectType):
            audience_now (int | Unset):
            automation_id (str | Unset):
            campaign_id (str | Unset):
            decided_at (datetime.datetime | Unset):
            decided_by (str | Unset):
            note (str | Unset):
            page (ApprovalPageSnapshot | Unset):
            schedule_at (datetime.datetime | Unset):
            sent_at (datetime.datetime | Unset):
            transactional_template_id (str | Unset):
            website (ApprovalWebsiteSnapshot | Unset):
            website_page_id (str | Unset):
            website_post_id (str | Unset):
     """

    audience: int
    author: ApprovalPerson
    channel: ApprovalListItemChannel
    id: str
    name: str
    requested_at: datetime.datetime
    route: str
    rules: str
    schedule_mode: ApprovalListItemScheduleMode
    status: ApprovalListItemStatus
    subject_type: ApprovalListItemSubjectType
    audience_now: int | Unset = UNSET
    automation_id: str | Unset = UNSET
    campaign_id: str | Unset = UNSET
    decided_at: datetime.datetime | Unset = UNSET
    decided_by: str | Unset = UNSET
    note: str | Unset = UNSET
    page: ApprovalPageSnapshot | Unset = UNSET
    schedule_at: datetime.datetime | Unset = UNSET
    sent_at: datetime.datetime | Unset = UNSET
    transactional_template_id: str | Unset = UNSET
    website: ApprovalWebsiteSnapshot | Unset = UNSET
    website_page_id: str | Unset = UNSET
    website_post_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.approval_page_snapshot import ApprovalPageSnapshot # noqa: PLC0415
        from ..models.approval_person import ApprovalPerson # noqa: PLC0415
        from ..models.approval_website_snapshot import ApprovalWebsiteSnapshot # noqa: PLC0415
        audience = self.audience

        author = self.author.to_dict()

        channel = self.channel.value

        id = self.id

        name = self.name

        requested_at = self.requested_at.isoformat()

        route = self.route

        rules = self.rules

        schedule_mode = self.schedule_mode.value

        status = self.status.value

        subject_type = self.subject_type.value

        audience_now = self.audience_now

        automation_id = self.automation_id

        campaign_id = self.campaign_id

        decided_at: str | Unset = UNSET
        if not isinstance(self.decided_at, Unset):
            decided_at = self.decided_at.isoformat()

        decided_by = self.decided_by

        note = self.note

        page: dict[str, Any] | Unset = UNSET
        if not isinstance(self.page, Unset):
            page = self.page.to_dict()

        schedule_at: str | Unset = UNSET
        if not isinstance(self.schedule_at, Unset):
            schedule_at = self.schedule_at.isoformat()

        sent_at: str | Unset = UNSET
        if not isinstance(self.sent_at, Unset):
            sent_at = self.sent_at.isoformat()

        transactional_template_id = self.transactional_template_id

        website: dict[str, Any] | Unset = UNSET
        if not isinstance(self.website, Unset):
            website = self.website.to_dict()

        website_page_id = self.website_page_id

        website_post_id = self.website_post_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "audience": audience,
            "author": author,
            "channel": channel,
            "id": id,
            "name": name,
            "requestedAt": requested_at,
            "route": route,
            "rules": rules,
            "scheduleMode": schedule_mode,
            "status": status,
            "subjectType": subject_type,
        })
        if audience_now is not UNSET:
            field_dict["audienceNow"] = audience_now
        if automation_id is not UNSET:
            field_dict["automationId"] = automation_id
        if campaign_id is not UNSET:
            field_dict["campaignId"] = campaign_id
        if decided_at is not UNSET:
            field_dict["decidedAt"] = decided_at
        if decided_by is not UNSET:
            field_dict["decidedBy"] = decided_by
        if note is not UNSET:
            field_dict["note"] = note
        if page is not UNSET:
            field_dict["page"] = page
        if schedule_at is not UNSET:
            field_dict["scheduleAt"] = schedule_at
        if sent_at is not UNSET:
            field_dict["sentAt"] = sent_at
        if transactional_template_id is not UNSET:
            field_dict["transactionalTemplateId"] = transactional_template_id
        if website is not UNSET:
            field_dict["website"] = website
        if website_page_id is not UNSET:
            field_dict["websitePageId"] = website_page_id
        if website_post_id is not UNSET:
            field_dict["websitePostId"] = website_post_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.approval_page_snapshot import ApprovalPageSnapshot # noqa: PLC0415
        from ..models.approval_person import ApprovalPerson # noqa: PLC0415
        from ..models.approval_website_snapshot import ApprovalWebsiteSnapshot # noqa: PLC0415
        d = dict(src_dict)
        audience = d.pop("audience")

        author = ApprovalPerson.from_dict(d.pop("author"))




        channel = ApprovalListItemChannel(d.pop("channel"))




        id = d.pop("id")

        name = d.pop("name")

        requested_at = datetime.datetime.fromisoformat(d.pop("requestedAt"))




        route = d.pop("route")

        rules = d.pop("rules")

        schedule_mode = ApprovalListItemScheduleMode(d.pop("scheduleMode"))




        status = ApprovalListItemStatus(d.pop("status"))




        subject_type = ApprovalListItemSubjectType(d.pop("subjectType"))




        audience_now = d.pop("audienceNow", UNSET)

        automation_id = d.pop("automationId", UNSET)

        campaign_id = d.pop("campaignId", UNSET)

        _decided_at = d.pop("decidedAt", UNSET)
        decided_at: datetime.datetime | Unset
        if isinstance(_decided_at,  Unset):
            decided_at = UNSET
        else:
            decided_at = datetime.datetime.fromisoformat(_decided_at)




        decided_by = d.pop("decidedBy", UNSET)

        note = d.pop("note", UNSET)

        _page = d.pop("page", UNSET)
        page: ApprovalPageSnapshot | Unset
        if isinstance(_page,  Unset):
            page = UNSET
        else:
            page = ApprovalPageSnapshot.from_dict(_page)




        _schedule_at = d.pop("scheduleAt", UNSET)
        schedule_at: datetime.datetime | Unset
        if isinstance(_schedule_at,  Unset):
            schedule_at = UNSET
        else:
            schedule_at = datetime.datetime.fromisoformat(_schedule_at)




        _sent_at = d.pop("sentAt", UNSET)
        sent_at: datetime.datetime | Unset
        if isinstance(_sent_at,  Unset):
            sent_at = UNSET
        else:
            sent_at = datetime.datetime.fromisoformat(_sent_at)




        transactional_template_id = d.pop("transactionalTemplateId", UNSET)

        _website = d.pop("website", UNSET)
        website: ApprovalWebsiteSnapshot | Unset
        if isinstance(_website,  Unset):
            website = UNSET
        else:
            website = ApprovalWebsiteSnapshot.from_dict(_website)




        website_page_id = d.pop("websitePageId", UNSET)

        website_post_id = d.pop("websitePostId", UNSET)

        approval_list_item = cls(
            audience=audience,
            author=author,
            channel=channel,
            id=id,
            name=name,
            requested_at=requested_at,
            route=route,
            rules=rules,
            schedule_mode=schedule_mode,
            status=status,
            subject_type=subject_type,
            audience_now=audience_now,
            automation_id=automation_id,
            campaign_id=campaign_id,
            decided_at=decided_at,
            decided_by=decided_by,
            note=note,
            page=page,
            schedule_at=schedule_at,
            sent_at=sent_at,
            transactional_template_id=transactional_template_id,
            website=website,
            website_page_id=website_page_id,
            website_post_id=website_post_id,
        )

        return approval_list_item

