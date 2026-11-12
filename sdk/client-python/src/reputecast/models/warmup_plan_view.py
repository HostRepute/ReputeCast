from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.warmup_plan_view_status import WarmupPlanViewStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.warmup_cap_hold import WarmupCapHold
  from ..models.warmup_log_entry import WarmupLogEntry
  from ..models.warmup_mailbox_cap import WarmupMailboxCap
  from ..models.warmup_ramp_day import WarmupRampDay
  from ..models.warmup_row import WarmupRow
  from ..models.warmup_settings import WarmupSettings
  from ..models.warmup_slowdown import WarmupSlowdown
  from ..models.warmup_step import WarmupStep





T = TypeVar("T", bound="WarmupPlanView")



@_attrs_define
class WarmupPlanView:
    """ 
        Attributes:
            calendar_days (int):
            count_share (float):
            counted (int):
            created_by (str):
            day (int):
            days (int):
            finishes_on (datetime.datetime):
            healthy_days (int):
            healthy_streak (int):
            holds (list[WarmupCapHold]):
            id (str):
            ip (str):
            log (list[WarmupLogEntry]):
            mailbox_caps (list[WarmupMailboxCap]):
            min_sample (int):
            need_today (int):
            overflow_today (int):
            progress (float): Percent of the plan's days counted
            provider (str):
            provider_id (str):
            ramp (list[WarmupRampDay]):
            rate_today (float):
            repeats_tomorrow (bool):
            rows (list[WarmupRow]):
            scheduled_today (int):
            sent_today (int):
            sent_total (int):
            settings (WarmupSettings):
            short (str):
            slowdowns (int):
            started_on (datetime.datetime):
            status (WarmupPlanViewStatus):
            status_detail (str):
            today_cap (int):
            transactional_over_cap (int):
            bounce_rate (float | Unset):
            complaint_rate (float | Unset):
            held (int | Unset):
            last_slowdown (WarmupSlowdown | Unset):
            next_rate (WarmupStep | Unset):
            overflow_to (str | Unset):
     """

    calendar_days: int
    count_share: float
    counted: int
    created_by: str
    day: int
    days: int
    finishes_on: datetime.datetime
    healthy_days: int
    healthy_streak: int
    holds: list[WarmupCapHold]
    id: str
    ip: str
    log: list[WarmupLogEntry]
    mailbox_caps: list[WarmupMailboxCap]
    min_sample: int
    need_today: int
    overflow_today: int
    progress: float
    provider: str
    provider_id: str
    ramp: list[WarmupRampDay]
    rate_today: float
    repeats_tomorrow: bool
    rows: list[WarmupRow]
    scheduled_today: int
    sent_today: int
    sent_total: int
    settings: WarmupSettings
    short: str
    slowdowns: int
    started_on: datetime.datetime
    status: WarmupPlanViewStatus
    status_detail: str
    today_cap: int
    transactional_over_cap: int
    bounce_rate: float | Unset = UNSET
    complaint_rate: float | Unset = UNSET
    held: int | Unset = UNSET
    last_slowdown: WarmupSlowdown | Unset = UNSET
    next_rate: WarmupStep | Unset = UNSET
    overflow_to: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.warmup_cap_hold import WarmupCapHold # noqa: PLC0415
        from ..models.warmup_log_entry import WarmupLogEntry # noqa: PLC0415
        from ..models.warmup_mailbox_cap import WarmupMailboxCap # noqa: PLC0415
        from ..models.warmup_ramp_day import WarmupRampDay # noqa: PLC0415
        from ..models.warmup_row import WarmupRow # noqa: PLC0415
        from ..models.warmup_settings import WarmupSettings # noqa: PLC0415
        from ..models.warmup_slowdown import WarmupSlowdown # noqa: PLC0415
        from ..models.warmup_step import WarmupStep # noqa: PLC0415
        calendar_days = self.calendar_days

        count_share = self.count_share

        counted = self.counted

        created_by = self.created_by

        day = self.day

        days = self.days

        finishes_on = self.finishes_on.isoformat()

        healthy_days = self.healthy_days

        healthy_streak = self.healthy_streak

        holds = []
        for holds_item_data in self.holds:
            holds_item = holds_item_data.to_dict()
            holds.append(holds_item)



        id = self.id

        ip = self.ip

        log = []
        for log_item_data in self.log:
            log_item = log_item_data.to_dict()
            log.append(log_item)



        mailbox_caps = []
        for mailbox_caps_item_data in self.mailbox_caps:
            mailbox_caps_item = mailbox_caps_item_data.to_dict()
            mailbox_caps.append(mailbox_caps_item)



        min_sample = self.min_sample

        need_today = self.need_today

        overflow_today = self.overflow_today

        progress = self.progress

        provider = self.provider

        provider_id = self.provider_id

        ramp = []
        for ramp_item_data in self.ramp:
            ramp_item = ramp_item_data.to_dict()
            ramp.append(ramp_item)



        rate_today = self.rate_today

        repeats_tomorrow = self.repeats_tomorrow

        rows = []
        for rows_item_data in self.rows:
            rows_item = rows_item_data.to_dict()
            rows.append(rows_item)



        scheduled_today = self.scheduled_today

        sent_today = self.sent_today

        sent_total = self.sent_total

        settings = self.settings.to_dict()

        short = self.short

        slowdowns = self.slowdowns

        started_on = self.started_on.isoformat()

        status = self.status.value

        status_detail = self.status_detail

        today_cap = self.today_cap

        transactional_over_cap = self.transactional_over_cap

        bounce_rate = self.bounce_rate

        complaint_rate = self.complaint_rate

        held = self.held

        last_slowdown: dict[str, Any] | Unset = UNSET
        if not isinstance(self.last_slowdown, Unset):
            last_slowdown = self.last_slowdown.to_dict()

        next_rate: dict[str, Any] | Unset = UNSET
        if not isinstance(self.next_rate, Unset):
            next_rate = self.next_rate.to_dict()

        overflow_to = self.overflow_to


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "calendarDays": calendar_days,
            "countShare": count_share,
            "counted": counted,
            "createdBy": created_by,
            "day": day,
            "days": days,
            "finishesOn": finishes_on,
            "healthyDays": healthy_days,
            "healthyStreak": healthy_streak,
            "holds": holds,
            "id": id,
            "ip": ip,
            "log": log,
            "mailboxCaps": mailbox_caps,
            "minSample": min_sample,
            "needToday": need_today,
            "overflowToday": overflow_today,
            "progress": progress,
            "provider": provider,
            "providerId": provider_id,
            "ramp": ramp,
            "rateToday": rate_today,
            "repeatsTomorrow": repeats_tomorrow,
            "rows": rows,
            "scheduledToday": scheduled_today,
            "sentToday": sent_today,
            "sentTotal": sent_total,
            "settings": settings,
            "short": short,
            "slowdowns": slowdowns,
            "startedOn": started_on,
            "status": status,
            "statusDetail": status_detail,
            "todayCap": today_cap,
            "transactionalOverCap": transactional_over_cap,
        })
        if bounce_rate is not UNSET:
            field_dict["bounceRate"] = bounce_rate
        if complaint_rate is not UNSET:
            field_dict["complaintRate"] = complaint_rate
        if held is not UNSET:
            field_dict["held"] = held
        if last_slowdown is not UNSET:
            field_dict["lastSlowdown"] = last_slowdown
        if next_rate is not UNSET:
            field_dict["nextRate"] = next_rate
        if overflow_to is not UNSET:
            field_dict["overflowTo"] = overflow_to

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.warmup_cap_hold import WarmupCapHold # noqa: PLC0415
        from ..models.warmup_log_entry import WarmupLogEntry # noqa: PLC0415
        from ..models.warmup_mailbox_cap import WarmupMailboxCap # noqa: PLC0415
        from ..models.warmup_ramp_day import WarmupRampDay # noqa: PLC0415
        from ..models.warmup_row import WarmupRow # noqa: PLC0415
        from ..models.warmup_settings import WarmupSettings # noqa: PLC0415
        from ..models.warmup_slowdown import WarmupSlowdown # noqa: PLC0415
        from ..models.warmup_step import WarmupStep # noqa: PLC0415
        d = dict(src_dict)
        calendar_days = d.pop("calendarDays")

        count_share = d.pop("countShare")

        counted = d.pop("counted")

        created_by = d.pop("createdBy")

        day = d.pop("day")

        days = d.pop("days")

        finishes_on = datetime.datetime.fromisoformat(d.pop("finishesOn"))




        healthy_days = d.pop("healthyDays")

        healthy_streak = d.pop("healthyStreak")

        holds = []
        _holds = d.pop("holds")
        for holds_item_data in (_holds):
            holds_item = WarmupCapHold.from_dict(holds_item_data)



            holds.append(holds_item)


        id = d.pop("id")

        ip = d.pop("ip")

        log = []
        _log = d.pop("log")
        for log_item_data in (_log):
            log_item = WarmupLogEntry.from_dict(log_item_data)



            log.append(log_item)


        mailbox_caps = []
        _mailbox_caps = d.pop("mailboxCaps")
        for mailbox_caps_item_data in (_mailbox_caps):
            mailbox_caps_item = WarmupMailboxCap.from_dict(mailbox_caps_item_data)



            mailbox_caps.append(mailbox_caps_item)


        min_sample = d.pop("minSample")

        need_today = d.pop("needToday")

        overflow_today = d.pop("overflowToday")

        progress = d.pop("progress")

        provider = d.pop("provider")

        provider_id = d.pop("providerId")

        ramp = []
        _ramp = d.pop("ramp")
        for ramp_item_data in (_ramp):
            ramp_item = WarmupRampDay.from_dict(ramp_item_data)



            ramp.append(ramp_item)


        rate_today = d.pop("rateToday")

        repeats_tomorrow = d.pop("repeatsTomorrow")

        rows = []
        _rows = d.pop("rows")
        for rows_item_data in (_rows):
            rows_item = WarmupRow.from_dict(rows_item_data)



            rows.append(rows_item)


        scheduled_today = d.pop("scheduledToday")

        sent_today = d.pop("sentToday")

        sent_total = d.pop("sentTotal")

        settings = WarmupSettings.from_dict(d.pop("settings"))




        short = d.pop("short")

        slowdowns = d.pop("slowdowns")

        started_on = datetime.datetime.fromisoformat(d.pop("startedOn"))




        status = WarmupPlanViewStatus(d.pop("status"))




        status_detail = d.pop("statusDetail")

        today_cap = d.pop("todayCap")

        transactional_over_cap = d.pop("transactionalOverCap")

        bounce_rate = d.pop("bounceRate", UNSET)

        complaint_rate = d.pop("complaintRate", UNSET)

        held = d.pop("held", UNSET)

        _last_slowdown = d.pop("lastSlowdown", UNSET)
        last_slowdown: WarmupSlowdown | Unset
        if isinstance(_last_slowdown,  Unset):
            last_slowdown = UNSET
        else:
            last_slowdown = WarmupSlowdown.from_dict(_last_slowdown)




        _next_rate = d.pop("nextRate", UNSET)
        next_rate: WarmupStep | Unset
        if isinstance(_next_rate,  Unset):
            next_rate = UNSET
        else:
            next_rate = WarmupStep.from_dict(_next_rate)




        overflow_to = d.pop("overflowTo", UNSET)

        warmup_plan_view = cls(
            calendar_days=calendar_days,
            count_share=count_share,
            counted=counted,
            created_by=created_by,
            day=day,
            days=days,
            finishes_on=finishes_on,
            healthy_days=healthy_days,
            healthy_streak=healthy_streak,
            holds=holds,
            id=id,
            ip=ip,
            log=log,
            mailbox_caps=mailbox_caps,
            min_sample=min_sample,
            need_today=need_today,
            overflow_today=overflow_today,
            progress=progress,
            provider=provider,
            provider_id=provider_id,
            ramp=ramp,
            rate_today=rate_today,
            repeats_tomorrow=repeats_tomorrow,
            rows=rows,
            scheduled_today=scheduled_today,
            sent_today=sent_today,
            sent_total=sent_total,
            settings=settings,
            short=short,
            slowdowns=slowdowns,
            started_on=started_on,
            status=status,
            status_detail=status_detail,
            today_cap=today_cap,
            transactional_over_cap=transactional_over_cap,
            bounce_rate=bounce_rate,
            complaint_rate=complaint_rate,
            held=held,
            last_slowdown=last_slowdown,
            next_rate=next_rate,
            overflow_to=overflow_to,
        )

        return warmup_plan_view

