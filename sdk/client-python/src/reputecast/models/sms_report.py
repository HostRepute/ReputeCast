from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_report_phase import SmsReportPhase
from ..models.sms_report_schedule_mode import SmsReportScheduleMode
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.sms_click_link import SmsClickLink
  from ..models.sms_country_report import SmsCountryReport
  from ..models.sms_failure_reason import SmsFailureReason
  from ..models.sms_held import SmsHeld
  from ..models.sms_outage_view import SmsOutageView
  from ..models.sms_pause_view import SmsPauseView
  from ..models.sms_reply import SmsReply
  from ..models.sms_report_reply_counts import SmsReportReplyCounts
  from ..models.sms_report_tiles import SmsReportTiles
  from ..models.sms_sender_report import SmsSenderReport
  from ..models.sms_time_point import SmsTimePoint
  from ..models.sms_timeline_sample import SmsTimelineSample
  from ..models.sms_zone_step import SmsZoneStep





T = TypeVar("T", bound="SmsReport")



@_attrs_define
class SmsReport:
    """ 
        Attributes:
            campaign_id (str):
            can_pause (bool):
            can_resume (bool):
            countries (list[SmsCountryReport]):
            failures (list[SmsFailureReason]):
            held (SmsHeld):
            helpers (list[str]):
            links (list[SmsClickLink]):
            name (str):
            phase (SmsReportPhase):
            points (list[SmsTimePoint]):
            receipts_within (float):
            recipients (int):
            replies (list[SmsReply]):
            reply_counts (SmsReportReplyCounts):
            schedule_mode (SmsReportScheduleMode):
            senders (list[SmsSenderReport]):
            short_domain (str):
            status (str):
            steps (list[SmsZoneStep]):
            tiles (SmsReportTiles):
            timezone (str):
            window (str):
            completed_at (datetime.datetime | Unset):
            local_time (str | Unset):
            outage (SmsOutageView | Unset):
            pause (SmsPauseView | Unset):
            sample (SmsTimelineSample | Unset):
            scheduled_at (datetime.datetime | Unset):
            started_at (datetime.datetime | Unset):
     """

    campaign_id: str
    can_pause: bool
    can_resume: bool
    countries: list[SmsCountryReport]
    failures: list[SmsFailureReason]
    held: SmsHeld
    helpers: list[str]
    links: list[SmsClickLink]
    name: str
    phase: SmsReportPhase
    points: list[SmsTimePoint]
    receipts_within: float
    recipients: int
    replies: list[SmsReply]
    reply_counts: SmsReportReplyCounts
    schedule_mode: SmsReportScheduleMode
    senders: list[SmsSenderReport]
    short_domain: str
    status: str
    steps: list[SmsZoneStep]
    tiles: SmsReportTiles
    timezone: str
    window: str
    completed_at: datetime.datetime | Unset = UNSET
    local_time: str | Unset = UNSET
    outage: SmsOutageView | Unset = UNSET
    pause: SmsPauseView | Unset = UNSET
    sample: SmsTimelineSample | Unset = UNSET
    scheduled_at: datetime.datetime | Unset = UNSET
    started_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_click_link import SmsClickLink # noqa: PLC0415
        from ..models.sms_country_report import SmsCountryReport # noqa: PLC0415
        from ..models.sms_failure_reason import SmsFailureReason # noqa: PLC0415
        from ..models.sms_held import SmsHeld # noqa: PLC0415
        from ..models.sms_outage_view import SmsOutageView # noqa: PLC0415
        from ..models.sms_pause_view import SmsPauseView # noqa: PLC0415
        from ..models.sms_reply import SmsReply # noqa: PLC0415
        from ..models.sms_report_reply_counts import SmsReportReplyCounts # noqa: PLC0415
        from ..models.sms_report_tiles import SmsReportTiles # noqa: PLC0415
        from ..models.sms_sender_report import SmsSenderReport # noqa: PLC0415
        from ..models.sms_time_point import SmsTimePoint # noqa: PLC0415
        from ..models.sms_timeline_sample import SmsTimelineSample # noqa: PLC0415
        from ..models.sms_zone_step import SmsZoneStep # noqa: PLC0415
        campaign_id = self.campaign_id

        can_pause = self.can_pause

        can_resume = self.can_resume

        countries = []
        for countries_item_data in self.countries:
            countries_item = countries_item_data.to_dict()
            countries.append(countries_item)



        failures = []
        for failures_item_data in self.failures:
            failures_item = failures_item_data.to_dict()
            failures.append(failures_item)



        held = self.held.to_dict()

        helpers = self.helpers



        links = []
        for links_item_data in self.links:
            links_item = links_item_data.to_dict()
            links.append(links_item)



        name = self.name

        phase = self.phase.value

        points = []
        for points_item_data in self.points:
            points_item = points_item_data.to_dict()
            points.append(points_item)



        receipts_within = self.receipts_within

        recipients = self.recipients

        replies = []
        for replies_item_data in self.replies:
            replies_item = replies_item_data.to_dict()
            replies.append(replies_item)



        reply_counts = self.reply_counts.to_dict()

        schedule_mode = self.schedule_mode.value

        senders = []
        for senders_item_data in self.senders:
            senders_item = senders_item_data.to_dict()
            senders.append(senders_item)



        short_domain = self.short_domain

        status = self.status

        steps = []
        for steps_item_data in self.steps:
            steps_item = steps_item_data.to_dict()
            steps.append(steps_item)



        tiles = self.tiles.to_dict()

        timezone = self.timezone

        window = self.window

        completed_at: str | Unset = UNSET
        if not isinstance(self.completed_at, Unset):
            completed_at = self.completed_at.isoformat()

        local_time = self.local_time

        outage: dict[str, Any] | Unset = UNSET
        if not isinstance(self.outage, Unset):
            outage = self.outage.to_dict()

        pause: dict[str, Any] | Unset = UNSET
        if not isinstance(self.pause, Unset):
            pause = self.pause.to_dict()

        sample: dict[str, Any] | Unset = UNSET
        if not isinstance(self.sample, Unset):
            sample = self.sample.to_dict()

        scheduled_at: str | Unset = UNSET
        if not isinstance(self.scheduled_at, Unset):
            scheduled_at = self.scheduled_at.isoformat()

        started_at: str | Unset = UNSET
        if not isinstance(self.started_at, Unset):
            started_at = self.started_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "campaignId": campaign_id,
            "canPause": can_pause,
            "canResume": can_resume,
            "countries": countries,
            "failures": failures,
            "held": held,
            "helpers": helpers,
            "links": links,
            "name": name,
            "phase": phase,
            "points": points,
            "receiptsWithin": receipts_within,
            "recipients": recipients,
            "replies": replies,
            "replyCounts": reply_counts,
            "scheduleMode": schedule_mode,
            "senders": senders,
            "shortDomain": short_domain,
            "status": status,
            "steps": steps,
            "tiles": tiles,
            "timezone": timezone,
            "window": window,
        })
        if completed_at is not UNSET:
            field_dict["completedAt"] = completed_at
        if local_time is not UNSET:
            field_dict["localTime"] = local_time
        if outage is not UNSET:
            field_dict["outage"] = outage
        if pause is not UNSET:
            field_dict["pause"] = pause
        if sample is not UNSET:
            field_dict["sample"] = sample
        if scheduled_at is not UNSET:
            field_dict["scheduledAt"] = scheduled_at
        if started_at is not UNSET:
            field_dict["startedAt"] = started_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_click_link import SmsClickLink # noqa: PLC0415
        from ..models.sms_country_report import SmsCountryReport # noqa: PLC0415
        from ..models.sms_failure_reason import SmsFailureReason # noqa: PLC0415
        from ..models.sms_held import SmsHeld # noqa: PLC0415
        from ..models.sms_outage_view import SmsOutageView # noqa: PLC0415
        from ..models.sms_pause_view import SmsPauseView # noqa: PLC0415
        from ..models.sms_reply import SmsReply # noqa: PLC0415
        from ..models.sms_report_reply_counts import SmsReportReplyCounts # noqa: PLC0415
        from ..models.sms_report_tiles import SmsReportTiles # noqa: PLC0415
        from ..models.sms_sender_report import SmsSenderReport # noqa: PLC0415
        from ..models.sms_time_point import SmsTimePoint # noqa: PLC0415
        from ..models.sms_timeline_sample import SmsTimelineSample # noqa: PLC0415
        from ..models.sms_zone_step import SmsZoneStep # noqa: PLC0415
        d = dict(src_dict)
        campaign_id = d.pop("campaignId")

        can_pause = d.pop("canPause")

        can_resume = d.pop("canResume")

        countries = []
        _countries = d.pop("countries")
        for countries_item_data in (_countries):
            countries_item = SmsCountryReport.from_dict(countries_item_data)



            countries.append(countries_item)


        failures = []
        _failures = d.pop("failures")
        for failures_item_data in (_failures):
            failures_item = SmsFailureReason.from_dict(failures_item_data)



            failures.append(failures_item)


        held = SmsHeld.from_dict(d.pop("held"))




        helpers = cast(list[str], d.pop("helpers"))


        links = []
        _links = d.pop("links")
        for links_item_data in (_links):
            links_item = SmsClickLink.from_dict(links_item_data)



            links.append(links_item)


        name = d.pop("name")

        phase = SmsReportPhase(d.pop("phase"))




        points = []
        _points = d.pop("points")
        for points_item_data in (_points):
            points_item = SmsTimePoint.from_dict(points_item_data)



            points.append(points_item)


        receipts_within = d.pop("receiptsWithin")

        recipients = d.pop("recipients")

        replies = []
        _replies = d.pop("replies")
        for replies_item_data in (_replies):
            replies_item = SmsReply.from_dict(replies_item_data)



            replies.append(replies_item)


        reply_counts = SmsReportReplyCounts.from_dict(d.pop("replyCounts"))




        schedule_mode = SmsReportScheduleMode(d.pop("scheduleMode"))




        senders = []
        _senders = d.pop("senders")
        for senders_item_data in (_senders):
            senders_item = SmsSenderReport.from_dict(senders_item_data)



            senders.append(senders_item)


        short_domain = d.pop("shortDomain")

        status = d.pop("status")

        steps = []
        _steps = d.pop("steps")
        for steps_item_data in (_steps):
            steps_item = SmsZoneStep.from_dict(steps_item_data)



            steps.append(steps_item)


        tiles = SmsReportTiles.from_dict(d.pop("tiles"))




        timezone = d.pop("timezone")

        window = d.pop("window")

        _completed_at = d.pop("completedAt", UNSET)
        completed_at: datetime.datetime | Unset
        if isinstance(_completed_at,  Unset):
            completed_at = UNSET
        else:
            completed_at = datetime.datetime.fromisoformat(_completed_at)




        local_time = d.pop("localTime", UNSET)

        _outage = d.pop("outage", UNSET)
        outage: SmsOutageView | Unset
        if isinstance(_outage,  Unset):
            outage = UNSET
        else:
            outage = SmsOutageView.from_dict(_outage)




        _pause = d.pop("pause", UNSET)
        pause: SmsPauseView | Unset
        if isinstance(_pause,  Unset):
            pause = UNSET
        else:
            pause = SmsPauseView.from_dict(_pause)




        _sample = d.pop("sample", UNSET)
        sample: SmsTimelineSample | Unset
        if isinstance(_sample,  Unset):
            sample = UNSET
        else:
            sample = SmsTimelineSample.from_dict(_sample)




        _scheduled_at = d.pop("scheduledAt", UNSET)
        scheduled_at: datetime.datetime | Unset
        if isinstance(_scheduled_at,  Unset):
            scheduled_at = UNSET
        else:
            scheduled_at = datetime.datetime.fromisoformat(_scheduled_at)




        _started_at = d.pop("startedAt", UNSET)
        started_at: datetime.datetime | Unset
        if isinstance(_started_at,  Unset):
            started_at = UNSET
        else:
            started_at = datetime.datetime.fromisoformat(_started_at)




        sms_report = cls(
            campaign_id=campaign_id,
            can_pause=can_pause,
            can_resume=can_resume,
            countries=countries,
            failures=failures,
            held=held,
            helpers=helpers,
            links=links,
            name=name,
            phase=phase,
            points=points,
            receipts_within=receipts_within,
            recipients=recipients,
            replies=replies,
            reply_counts=reply_counts,
            schedule_mode=schedule_mode,
            senders=senders,
            short_domain=short_domain,
            status=status,
            steps=steps,
            tiles=tiles,
            timezone=timezone,
            window=window,
            completed_at=completed_at,
            local_time=local_time,
            outage=outage,
            pause=pause,
            sample=sample,
            scheduled_at=scheduled_at,
            started_at=started_at,
        )

        return sms_report

