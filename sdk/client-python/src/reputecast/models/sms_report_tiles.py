from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.sms_provider_count import SmsProviderCount
  from ..models.sms_report_tiles_failed_by import SmsReportTilesFailedBy





T = TypeVar("T", bound="SmsReportTiles")



@_attrs_define
class SmsReportTiles:
    """ 
        Attributes:
            by_provider (list[SmsProviderCount]):
            click_pct (float):
            clicks (int):
            cost (float):
            delivered (int):
            delivered_pct (float):
            failed (int):
            failed_by (SmsReportTilesFailedBy):
            held (int):
            machine (int):
            opt_out_pct (float):
            opt_outs (int):
            opt_outs_by_keyword (bool):
            per_text (float):
            planned (float):
            queued (int):
            sent (int):
            skipped (int):
            waiting (int):
            within_budget (bool):
     """

    by_provider: list[SmsProviderCount]
    click_pct: float
    clicks: int
    cost: float
    delivered: int
    delivered_pct: float
    failed: int
    failed_by: SmsReportTilesFailedBy
    held: int
    machine: int
    opt_out_pct: float
    opt_outs: int
    opt_outs_by_keyword: bool
    per_text: float
    planned: float
    queued: int
    sent: int
    skipped: int
    waiting: int
    within_budget: bool





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_provider_count import SmsProviderCount # noqa: PLC0415
        from ..models.sms_report_tiles_failed_by import SmsReportTilesFailedBy # noqa: PLC0415
        by_provider = []
        for by_provider_item_data in self.by_provider:
            by_provider_item = by_provider_item_data.to_dict()
            by_provider.append(by_provider_item)



        click_pct = self.click_pct

        clicks = self.clicks

        cost = self.cost

        delivered = self.delivered

        delivered_pct = self.delivered_pct

        failed = self.failed

        failed_by = self.failed_by.to_dict()

        held = self.held

        machine = self.machine

        opt_out_pct = self.opt_out_pct

        opt_outs = self.opt_outs

        opt_outs_by_keyword = self.opt_outs_by_keyword

        per_text = self.per_text

        planned = self.planned

        queued = self.queued

        sent = self.sent

        skipped = self.skipped

        waiting = self.waiting

        within_budget = self.within_budget


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "byProvider": by_provider,
            "clickPct": click_pct,
            "clicks": clicks,
            "cost": cost,
            "delivered": delivered,
            "deliveredPct": delivered_pct,
            "failed": failed,
            "failedBy": failed_by,
            "held": held,
            "machine": machine,
            "optOutPct": opt_out_pct,
            "optOuts": opt_outs,
            "optOutsByKeyword": opt_outs_by_keyword,
            "perText": per_text,
            "planned": planned,
            "queued": queued,
            "sent": sent,
            "skipped": skipped,
            "waiting": waiting,
            "withinBudget": within_budget,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_provider_count import SmsProviderCount # noqa: PLC0415
        from ..models.sms_report_tiles_failed_by import SmsReportTilesFailedBy # noqa: PLC0415
        d = dict(src_dict)
        by_provider = []
        _by_provider = d.pop("byProvider")
        for by_provider_item_data in (_by_provider):
            by_provider_item = SmsProviderCount.from_dict(by_provider_item_data)



            by_provider.append(by_provider_item)


        click_pct = d.pop("clickPct")

        clicks = d.pop("clicks")

        cost = d.pop("cost")

        delivered = d.pop("delivered")

        delivered_pct = d.pop("deliveredPct")

        failed = d.pop("failed")

        failed_by = SmsReportTilesFailedBy.from_dict(d.pop("failedBy"))




        held = d.pop("held")

        machine = d.pop("machine")

        opt_out_pct = d.pop("optOutPct")

        opt_outs = d.pop("optOuts")

        opt_outs_by_keyword = d.pop("optOutsByKeyword")

        per_text = d.pop("perText")

        planned = d.pop("planned")

        queued = d.pop("queued")

        sent = d.pop("sent")

        skipped = d.pop("skipped")

        waiting = d.pop("waiting")

        within_budget = d.pop("withinBudget")

        sms_report_tiles = cls(
            by_provider=by_provider,
            click_pct=click_pct,
            clicks=clicks,
            cost=cost,
            delivered=delivered,
            delivered_pct=delivered_pct,
            failed=failed,
            failed_by=failed_by,
            held=held,
            machine=machine,
            opt_out_pct=opt_out_pct,
            opt_outs=opt_outs,
            opt_outs_by_keyword=opt_outs_by_keyword,
            per_text=per_text,
            planned=planned,
            queued=queued,
            sent=sent,
            skipped=skipped,
            waiting=waiting,
            within_budget=within_budget,
        )

        return sms_report_tiles

