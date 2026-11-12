from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.approval_snapshot_summary_schedule_mode import ApprovalSnapshotSummaryScheduleMode
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ApprovalSnapshotSummary")



@_attrs_define
class ApprovalSnapshotSummary:
    """ 
        Attributes:
            audience_count (int):
            route_label (str):
            route_text (str):
            schedule_mode (ApprovalSnapshotSummaryScheduleMode):
            schedule_text (str):
            split_label (str):
            subject (str):
            version (int):
            schedule_at (datetime.datetime | Unset):
            timezone (str | Unset):
     """

    audience_count: int
    route_label: str
    route_text: str
    schedule_mode: ApprovalSnapshotSummaryScheduleMode
    schedule_text: str
    split_label: str
    subject: str
    version: int
    schedule_at: datetime.datetime | Unset = UNSET
    timezone: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        audience_count = self.audience_count

        route_label = self.route_label

        route_text = self.route_text

        schedule_mode = self.schedule_mode.value

        schedule_text = self.schedule_text

        split_label = self.split_label

        subject = self.subject

        version = self.version

        schedule_at: str | Unset = UNSET
        if not isinstance(self.schedule_at, Unset):
            schedule_at = self.schedule_at.isoformat()

        timezone = self.timezone


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "audienceCount": audience_count,
            "routeLabel": route_label,
            "routeText": route_text,
            "scheduleMode": schedule_mode,
            "scheduleText": schedule_text,
            "splitLabel": split_label,
            "subject": subject,
            "version": version,
        })
        if schedule_at is not UNSET:
            field_dict["scheduleAt"] = schedule_at
        if timezone is not UNSET:
            field_dict["timezone"] = timezone

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        audience_count = d.pop("audienceCount")

        route_label = d.pop("routeLabel")

        route_text = d.pop("routeText")

        schedule_mode = ApprovalSnapshotSummaryScheduleMode(d.pop("scheduleMode"))




        schedule_text = d.pop("scheduleText")

        split_label = d.pop("splitLabel")

        subject = d.pop("subject")

        version = d.pop("version")

        _schedule_at = d.pop("scheduleAt", UNSET)
        schedule_at: datetime.datetime | Unset
        if isinstance(_schedule_at,  Unset):
            schedule_at = UNSET
        else:
            schedule_at = datetime.datetime.fromisoformat(_schedule_at)




        timezone = d.pop("timezone", UNSET)

        approval_snapshot_summary = cls(
            audience_count=audience_count,
            route_label=route_label,
            route_text=route_text,
            schedule_mode=schedule_mode,
            schedule_text=schedule_text,
            split_label=split_label,
            subject=subject,
            version=version,
            schedule_at=schedule_at,
            timezone=timezone,
        )

        return approval_snapshot_summary

