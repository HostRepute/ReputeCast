from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.digest_detail_mode import DigestDetailMode
from ..models.digest_detail_source import DigestDetailSource
from ..models.digest_detail_state import DigestDetailState
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.digest_approvals import DigestApprovals
  from ..models.digest_audience import DigestAudience
  from ..models.digest_feed_status import DigestFeedStatus
  from ..models.digest_filters import DigestFilters
  from ..models.digest_next_issue import DigestNextIssue
  from ..models.digest_pause import DigestPause
  from ..models.digest_person import DigestPerson
  from ..models.digest_route import DigestRoute
  from ..models.digest_schedule import DigestSchedule
  from ..models.digest_sender import DigestSender
  from ..models.digest_site import DigestSite
  from ..models.digest_thresholds import DigestThresholds





T = TypeVar("T", bound="DigestDetail")



@_attrs_define
class DigestDetail:
    """ 
        Attributes:
            approvals (DigestApprovals):
            archive (bool):
            audience (DigestAudience):
            category_options (list[str]):
            created_at (datetime.datetime):
            edit_helpers (list[str]):
            feed (DigestFeedStatus):
            feed_url (str):
            filters (DigestFilters):
            id (str):
            late_hours (int):
            mode (DigestDetailMode):
            name (str):
            pause_helpers (list[str]):
            prepare_hours (int):
            route (DigestRoute):
            schedule (DigestSchedule):
            send_helpers (list[str]):
            sent_issues (int):
            source (DigestDetailSource):
            state (DigestDetailState):
            tag_options (list[str]):
            template_id (str):
            template_version (int):
            thresholds (DigestThresholds):
            updated_at (datetime.datetime):
            created_by (DigestPerson | Unset):
            first_on_at (datetime.datetime | Unset):
            first_sent_at (datetime.datetime | Unset):
            next_ (DigestNextIssue | Unset):
            next_send (datetime.datetime | Unset):
            owner (DigestPerson | Unset):
            owner_since (datetime.datetime | Unset):
            pause (DigestPause | Unset):
            sender (DigestSender | Unset):
            site (DigestSite | Unset):
            template_saved_at (datetime.datetime | Unset):
     """

    approvals: DigestApprovals
    archive: bool
    audience: DigestAudience
    category_options: list[str]
    created_at: datetime.datetime
    edit_helpers: list[str]
    feed: DigestFeedStatus
    feed_url: str
    filters: DigestFilters
    id: str
    late_hours: int
    mode: DigestDetailMode
    name: str
    pause_helpers: list[str]
    prepare_hours: int
    route: DigestRoute
    schedule: DigestSchedule
    send_helpers: list[str]
    sent_issues: int
    source: DigestDetailSource
    state: DigestDetailState
    tag_options: list[str]
    template_id: str
    template_version: int
    thresholds: DigestThresholds
    updated_at: datetime.datetime
    created_by: DigestPerson | Unset = UNSET
    first_on_at: datetime.datetime | Unset = UNSET
    first_sent_at: datetime.datetime | Unset = UNSET
    next_: DigestNextIssue | Unset = UNSET
    next_send: datetime.datetime | Unset = UNSET
    owner: DigestPerson | Unset = UNSET
    owner_since: datetime.datetime | Unset = UNSET
    pause: DigestPause | Unset = UNSET
    sender: DigestSender | Unset = UNSET
    site: DigestSite | Unset = UNSET
    template_saved_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.digest_approvals import DigestApprovals # noqa: PLC0415
        from ..models.digest_audience import DigestAudience # noqa: PLC0415
        from ..models.digest_feed_status import DigestFeedStatus # noqa: PLC0415
        from ..models.digest_filters import DigestFilters # noqa: PLC0415
        from ..models.digest_next_issue import DigestNextIssue # noqa: PLC0415
        from ..models.digest_pause import DigestPause # noqa: PLC0415
        from ..models.digest_person import DigestPerson # noqa: PLC0415
        from ..models.digest_route import DigestRoute # noqa: PLC0415
        from ..models.digest_schedule import DigestSchedule # noqa: PLC0415
        from ..models.digest_sender import DigestSender # noqa: PLC0415
        from ..models.digest_site import DigestSite # noqa: PLC0415
        from ..models.digest_thresholds import DigestThresholds # noqa: PLC0415
        approvals = self.approvals.to_dict()

        archive = self.archive

        audience = self.audience.to_dict()

        category_options = self.category_options



        created_at = self.created_at.isoformat()

        edit_helpers = self.edit_helpers



        feed = self.feed.to_dict()

        feed_url = self.feed_url

        filters = self.filters.to_dict()

        id = self.id

        late_hours = self.late_hours

        mode = self.mode.value

        name = self.name

        pause_helpers = self.pause_helpers



        prepare_hours = self.prepare_hours

        route = self.route.to_dict()

        schedule = self.schedule.to_dict()

        send_helpers = self.send_helpers



        sent_issues = self.sent_issues

        source = self.source.value

        state = self.state.value

        tag_options = self.tag_options



        template_id = self.template_id

        template_version = self.template_version

        thresholds = self.thresholds.to_dict()

        updated_at = self.updated_at.isoformat()

        created_by: dict[str, Any] | Unset = UNSET
        if not isinstance(self.created_by, Unset):
            created_by = self.created_by.to_dict()

        first_on_at: str | Unset = UNSET
        if not isinstance(self.first_on_at, Unset):
            first_on_at = self.first_on_at.isoformat()

        first_sent_at: str | Unset = UNSET
        if not isinstance(self.first_sent_at, Unset):
            first_sent_at = self.first_sent_at.isoformat()

        next_: dict[str, Any] | Unset = UNSET
        if not isinstance(self.next_, Unset):
            next_ = self.next_.to_dict()

        next_send: str | Unset = UNSET
        if not isinstance(self.next_send, Unset):
            next_send = self.next_send.isoformat()

        owner: dict[str, Any] | Unset = UNSET
        if not isinstance(self.owner, Unset):
            owner = self.owner.to_dict()

        owner_since: str | Unset = UNSET
        if not isinstance(self.owner_since, Unset):
            owner_since = self.owner_since.isoformat()

        pause: dict[str, Any] | Unset = UNSET
        if not isinstance(self.pause, Unset):
            pause = self.pause.to_dict()

        sender: dict[str, Any] | Unset = UNSET
        if not isinstance(self.sender, Unset):
            sender = self.sender.to_dict()

        site: dict[str, Any] | Unset = UNSET
        if not isinstance(self.site, Unset):
            site = self.site.to_dict()

        template_saved_at: str | Unset = UNSET
        if not isinstance(self.template_saved_at, Unset):
            template_saved_at = self.template_saved_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "approvals": approvals,
            "archive": archive,
            "audience": audience,
            "categoryOptions": category_options,
            "createdAt": created_at,
            "editHelpers": edit_helpers,
            "feed": feed,
            "feedUrl": feed_url,
            "filters": filters,
            "id": id,
            "lateHours": late_hours,
            "mode": mode,
            "name": name,
            "pauseHelpers": pause_helpers,
            "prepareHours": prepare_hours,
            "route": route,
            "schedule": schedule,
            "sendHelpers": send_helpers,
            "sentIssues": sent_issues,
            "source": source,
            "state": state,
            "tagOptions": tag_options,
            "templateId": template_id,
            "templateVersion": template_version,
            "thresholds": thresholds,
            "updatedAt": updated_at,
        })
        if created_by is not UNSET:
            field_dict["createdBy"] = created_by
        if first_on_at is not UNSET:
            field_dict["firstOnAt"] = first_on_at
        if first_sent_at is not UNSET:
            field_dict["firstSentAt"] = first_sent_at
        if next_ is not UNSET:
            field_dict["next"] = next_
        if next_send is not UNSET:
            field_dict["nextSend"] = next_send
        if owner is not UNSET:
            field_dict["owner"] = owner
        if owner_since is not UNSET:
            field_dict["ownerSince"] = owner_since
        if pause is not UNSET:
            field_dict["pause"] = pause
        if sender is not UNSET:
            field_dict["sender"] = sender
        if site is not UNSET:
            field_dict["site"] = site
        if template_saved_at is not UNSET:
            field_dict["templateSavedAt"] = template_saved_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.digest_approvals import DigestApprovals # noqa: PLC0415
        from ..models.digest_audience import DigestAudience # noqa: PLC0415
        from ..models.digest_feed_status import DigestFeedStatus # noqa: PLC0415
        from ..models.digest_filters import DigestFilters # noqa: PLC0415
        from ..models.digest_next_issue import DigestNextIssue # noqa: PLC0415
        from ..models.digest_pause import DigestPause # noqa: PLC0415
        from ..models.digest_person import DigestPerson # noqa: PLC0415
        from ..models.digest_route import DigestRoute # noqa: PLC0415
        from ..models.digest_schedule import DigestSchedule # noqa: PLC0415
        from ..models.digest_sender import DigestSender # noqa: PLC0415
        from ..models.digest_site import DigestSite # noqa: PLC0415
        from ..models.digest_thresholds import DigestThresholds # noqa: PLC0415
        d = dict(src_dict)
        approvals = DigestApprovals.from_dict(d.pop("approvals"))




        archive = d.pop("archive")

        audience = DigestAudience.from_dict(d.pop("audience"))




        category_options = cast(list[str], d.pop("categoryOptions"))


        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        edit_helpers = cast(list[str], d.pop("editHelpers"))


        feed = DigestFeedStatus.from_dict(d.pop("feed"))




        feed_url = d.pop("feedUrl")

        filters = DigestFilters.from_dict(d.pop("filters"))




        id = d.pop("id")

        late_hours = d.pop("lateHours")

        mode = DigestDetailMode(d.pop("mode"))




        name = d.pop("name")

        pause_helpers = cast(list[str], d.pop("pauseHelpers"))


        prepare_hours = d.pop("prepareHours")

        route = DigestRoute.from_dict(d.pop("route"))




        schedule = DigestSchedule.from_dict(d.pop("schedule"))




        send_helpers = cast(list[str], d.pop("sendHelpers"))


        sent_issues = d.pop("sentIssues")

        source = DigestDetailSource(d.pop("source"))




        state = DigestDetailState(d.pop("state"))




        tag_options = cast(list[str], d.pop("tagOptions"))


        template_id = d.pop("templateId")

        template_version = d.pop("templateVersion")

        thresholds = DigestThresholds.from_dict(d.pop("thresholds"))




        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))




        _created_by = d.pop("createdBy", UNSET)
        created_by: DigestPerson | Unset
        if isinstance(_created_by,  Unset):
            created_by = UNSET
        else:
            created_by = DigestPerson.from_dict(_created_by)




        _first_on_at = d.pop("firstOnAt", UNSET)
        first_on_at: datetime.datetime | Unset
        if isinstance(_first_on_at,  Unset):
            first_on_at = UNSET
        else:
            first_on_at = datetime.datetime.fromisoformat(_first_on_at)




        _first_sent_at = d.pop("firstSentAt", UNSET)
        first_sent_at: datetime.datetime | Unset
        if isinstance(_first_sent_at,  Unset):
            first_sent_at = UNSET
        else:
            first_sent_at = datetime.datetime.fromisoformat(_first_sent_at)




        _next_ = d.pop("next", UNSET)
        next_: DigestNextIssue | Unset
        if isinstance(_next_,  Unset):
            next_ = UNSET
        else:
            next_ = DigestNextIssue.from_dict(_next_)




        _next_send = d.pop("nextSend", UNSET)
        next_send: datetime.datetime | Unset
        if isinstance(_next_send,  Unset):
            next_send = UNSET
        else:
            next_send = datetime.datetime.fromisoformat(_next_send)




        _owner = d.pop("owner", UNSET)
        owner: DigestPerson | Unset
        if isinstance(_owner,  Unset):
            owner = UNSET
        else:
            owner = DigestPerson.from_dict(_owner)




        _owner_since = d.pop("ownerSince", UNSET)
        owner_since: datetime.datetime | Unset
        if isinstance(_owner_since,  Unset):
            owner_since = UNSET
        else:
            owner_since = datetime.datetime.fromisoformat(_owner_since)




        _pause = d.pop("pause", UNSET)
        pause: DigestPause | Unset
        if isinstance(_pause,  Unset):
            pause = UNSET
        else:
            pause = DigestPause.from_dict(_pause)




        _sender = d.pop("sender", UNSET)
        sender: DigestSender | Unset
        if isinstance(_sender,  Unset):
            sender = UNSET
        else:
            sender = DigestSender.from_dict(_sender)




        _site = d.pop("site", UNSET)
        site: DigestSite | Unset
        if isinstance(_site,  Unset):
            site = UNSET
        else:
            site = DigestSite.from_dict(_site)




        _template_saved_at = d.pop("templateSavedAt", UNSET)
        template_saved_at: datetime.datetime | Unset
        if isinstance(_template_saved_at,  Unset):
            template_saved_at = UNSET
        else:
            template_saved_at = datetime.datetime.fromisoformat(_template_saved_at)




        digest_detail = cls(
            approvals=approvals,
            archive=archive,
            audience=audience,
            category_options=category_options,
            created_at=created_at,
            edit_helpers=edit_helpers,
            feed=feed,
            feed_url=feed_url,
            filters=filters,
            id=id,
            late_hours=late_hours,
            mode=mode,
            name=name,
            pause_helpers=pause_helpers,
            prepare_hours=prepare_hours,
            route=route,
            schedule=schedule,
            send_helpers=send_helpers,
            sent_issues=sent_issues,
            source=source,
            state=state,
            tag_options=tag_options,
            template_id=template_id,
            template_version=template_version,
            thresholds=thresholds,
            updated_at=updated_at,
            created_by=created_by,
            first_on_at=first_on_at,
            first_sent_at=first_sent_at,
            next_=next_,
            next_send=next_send,
            owner=owner,
            owner_since=owner_since,
            pause=pause,
            sender=sender,
            site=site,
            template_saved_at=template_saved_at,
        )

        return digest_detail

