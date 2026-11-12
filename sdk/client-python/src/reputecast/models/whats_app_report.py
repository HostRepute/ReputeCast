from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.whats_app_report_status import WhatsAppReportStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.whats_app_failure import WhatsAppFailure
  from ..models.whats_app_report_country import WhatsAppReportCountry
  from ..models.whats_app_report_skipped import WhatsAppReportSkipped





T = TypeVar("T", bound="WhatsAppReport")



@_attrs_define
class WhatsAppReport:
    """ 
        Attributes:
            buttons (int):
            by_country (list[WhatsAppReportCountry]):
            category (str):
            cost (float):
            delivered (int):
            delivered_pct (float):
            estimate (float):
            failed (int):
            failures (list[WhatsAppFailure]):
            held (int):
            id (str):
            name (str):
            opt_outs (int):
            planned (int):
            queued (int):
            rates_as_of (datetime.datetime):
            read (int):
            read_pct (float):
            replies (int):
            sent (int):
            sent_by (str):
            skipped (WhatsAppReportSkipped):
            status (WhatsAppReportStatus):
            template (str):
            sent_at (datetime.datetime | Unset):
     """

    buttons: int
    by_country: list[WhatsAppReportCountry]
    category: str
    cost: float
    delivered: int
    delivered_pct: float
    estimate: float
    failed: int
    failures: list[WhatsAppFailure]
    held: int
    id: str
    name: str
    opt_outs: int
    planned: int
    queued: int
    rates_as_of: datetime.datetime
    read: int
    read_pct: float
    replies: int
    sent: int
    sent_by: str
    skipped: WhatsAppReportSkipped
    status: WhatsAppReportStatus
    template: str
    sent_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.whats_app_failure import WhatsAppFailure # noqa: PLC0415
        from ..models.whats_app_report_country import WhatsAppReportCountry # noqa: PLC0415
        from ..models.whats_app_report_skipped import WhatsAppReportSkipped # noqa: PLC0415
        buttons = self.buttons

        by_country = []
        for by_country_item_data in self.by_country:
            by_country_item = by_country_item_data.to_dict()
            by_country.append(by_country_item)



        category = self.category

        cost = self.cost

        delivered = self.delivered

        delivered_pct = self.delivered_pct

        estimate = self.estimate

        failed = self.failed

        failures = []
        for failures_item_data in self.failures:
            failures_item = failures_item_data.to_dict()
            failures.append(failures_item)



        held = self.held

        id = self.id

        name = self.name

        opt_outs = self.opt_outs

        planned = self.planned

        queued = self.queued

        rates_as_of = self.rates_as_of.isoformat()

        read = self.read

        read_pct = self.read_pct

        replies = self.replies

        sent = self.sent

        sent_by = self.sent_by

        skipped = self.skipped.to_dict()

        status = self.status.value

        template = self.template

        sent_at: str | Unset = UNSET
        if not isinstance(self.sent_at, Unset):
            sent_at = self.sent_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "buttons": buttons,
            "byCountry": by_country,
            "category": category,
            "cost": cost,
            "delivered": delivered,
            "deliveredPct": delivered_pct,
            "estimate": estimate,
            "failed": failed,
            "failures": failures,
            "held": held,
            "id": id,
            "name": name,
            "optOuts": opt_outs,
            "planned": planned,
            "queued": queued,
            "ratesAsOf": rates_as_of,
            "read": read,
            "readPct": read_pct,
            "replies": replies,
            "sent": sent,
            "sentBy": sent_by,
            "skipped": skipped,
            "status": status,
            "template": template,
        })
        if sent_at is not UNSET:
            field_dict["sentAt"] = sent_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.whats_app_failure import WhatsAppFailure # noqa: PLC0415
        from ..models.whats_app_report_country import WhatsAppReportCountry # noqa: PLC0415
        from ..models.whats_app_report_skipped import WhatsAppReportSkipped # noqa: PLC0415
        d = dict(src_dict)
        buttons = d.pop("buttons")

        by_country = []
        _by_country = d.pop("byCountry")
        for by_country_item_data in (_by_country):
            by_country_item = WhatsAppReportCountry.from_dict(by_country_item_data)



            by_country.append(by_country_item)


        category = d.pop("category")

        cost = d.pop("cost")

        delivered = d.pop("delivered")

        delivered_pct = d.pop("deliveredPct")

        estimate = d.pop("estimate")

        failed = d.pop("failed")

        failures = []
        _failures = d.pop("failures")
        for failures_item_data in (_failures):
            failures_item = WhatsAppFailure.from_dict(failures_item_data)



            failures.append(failures_item)


        held = d.pop("held")

        id = d.pop("id")

        name = d.pop("name")

        opt_outs = d.pop("optOuts")

        planned = d.pop("planned")

        queued = d.pop("queued")

        rates_as_of = datetime.datetime.fromisoformat(d.pop("ratesAsOf"))




        read = d.pop("read")

        read_pct = d.pop("readPct")

        replies = d.pop("replies")

        sent = d.pop("sent")

        sent_by = d.pop("sentBy")

        skipped = WhatsAppReportSkipped.from_dict(d.pop("skipped"))




        status = WhatsAppReportStatus(d.pop("status"))




        template = d.pop("template")

        _sent_at = d.pop("sentAt", UNSET)
        sent_at: datetime.datetime | Unset
        if isinstance(_sent_at,  Unset):
            sent_at = UNSET
        else:
            sent_at = datetime.datetime.fromisoformat(_sent_at)




        whats_app_report = cls(
            buttons=buttons,
            by_country=by_country,
            category=category,
            cost=cost,
            delivered=delivered,
            delivered_pct=delivered_pct,
            estimate=estimate,
            failed=failed,
            failures=failures,
            held=held,
            id=id,
            name=name,
            opt_outs=opt_outs,
            planned=planned,
            queued=queued,
            rates_as_of=rates_as_of,
            read=read,
            read_pct=read_pct,
            replies=replies,
            sent=sent,
            sent_by=sent_by,
            skipped=skipped,
            status=status,
            template=template,
            sent_at=sent_at,
        )

        return whats_app_report

