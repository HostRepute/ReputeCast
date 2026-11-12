from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.report_not_sent_readiness import ReportNotSentReadiness
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.report_campaign_ref import ReportCampaignRef





T = TypeVar("T", bound="ReportNotSent")



@_attrs_define
class ReportNotSent:
    """ 
        Attributes:
            status (str):
            last_report (ReportCampaignRef | Unset):
            readiness (ReportNotSentReadiness | Unset):
            scheduled_at (datetime.datetime | Unset):
            timezone (str | Unset):
     """

    status: str
    last_report: ReportCampaignRef | Unset = UNSET
    readiness: ReportNotSentReadiness | Unset = UNSET
    scheduled_at: datetime.datetime | Unset = UNSET
    timezone: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.report_campaign_ref import ReportCampaignRef # noqa: PLC0415
        status = self.status

        last_report: dict[str, Any] | Unset = UNSET
        if not isinstance(self.last_report, Unset):
            last_report = self.last_report.to_dict()

        readiness: str | Unset = UNSET
        if not isinstance(self.readiness, Unset):
            readiness = self.readiness.value


        scheduled_at: str | Unset = UNSET
        if not isinstance(self.scheduled_at, Unset):
            scheduled_at = self.scheduled_at.isoformat()

        timezone = self.timezone


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "status": status,
        })
        if last_report is not UNSET:
            field_dict["lastReport"] = last_report
        if readiness is not UNSET:
            field_dict["readiness"] = readiness
        if scheduled_at is not UNSET:
            field_dict["scheduledAt"] = scheduled_at
        if timezone is not UNSET:
            field_dict["timezone"] = timezone

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.report_campaign_ref import ReportCampaignRef # noqa: PLC0415
        d = dict(src_dict)
        status = d.pop("status")

        _last_report = d.pop("lastReport", UNSET)
        last_report: ReportCampaignRef | Unset
        if isinstance(_last_report,  Unset):
            last_report = UNSET
        else:
            last_report = ReportCampaignRef.from_dict(_last_report)




        _readiness = d.pop("readiness", UNSET)
        readiness: ReportNotSentReadiness | Unset
        if isinstance(_readiness,  Unset):
            readiness = UNSET
        else:
            readiness = ReportNotSentReadiness(_readiness)




        _scheduled_at = d.pop("scheduledAt", UNSET)
        scheduled_at: datetime.datetime | Unset
        if isinstance(_scheduled_at,  Unset):
            scheduled_at = UNSET
        else:
            scheduled_at = datetime.datetime.fromisoformat(_scheduled_at)




        timezone = d.pop("timezone", UNSET)

        report_not_sent = cls(
            status=status,
            last_report=last_report,
            readiness=readiness,
            scheduled_at=scheduled_at,
            timezone=timezone,
        )

        return report_not_sent

