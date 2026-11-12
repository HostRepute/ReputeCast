from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.warmup_row_metric import WarmupRowMetric
from ..models.warmup_row_reason import WarmupRowReason
from ..models.warmup_row_status import WarmupRowStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.warmup_campaign_use import WarmupCampaignUse





T = TypeVar("T", bound="WarmupRow")



@_attrs_define
class WarmupRow:
    """ 
        Attributes:
            bounced (int):
            cap (int):
            complained (int):
            date (datetime.datetime):
            day (int):
            delivered (int):
            eligible (bool):
            held (bool):
            judged (bool):
            need (int):
            repeat (bool):
            share (int):
            share_change (bool):
            status (WarmupRowStatus):
            tiers (list[str]):
            uses (list[WarmupCampaignUse]):
            bounce (float | Unset):
            complaint (float | Unset):
            healthy (int | Unset):
            metric (WarmupRowMetric | Unset):
            planned_cap (int | Unset):
            reason (WarmupRowReason | Unset):
            sent (int | Unset):
     """

    bounced: int
    cap: int
    complained: int
    date: datetime.datetime
    day: int
    delivered: int
    eligible: bool
    held: bool
    judged: bool
    need: int
    repeat: bool
    share: int
    share_change: bool
    status: WarmupRowStatus
    tiers: list[str]
    uses: list[WarmupCampaignUse]
    bounce: float | Unset = UNSET
    complaint: float | Unset = UNSET
    healthy: int | Unset = UNSET
    metric: WarmupRowMetric | Unset = UNSET
    planned_cap: int | Unset = UNSET
    reason: WarmupRowReason | Unset = UNSET
    sent: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.warmup_campaign_use import WarmupCampaignUse # noqa: PLC0415
        bounced = self.bounced

        cap = self.cap

        complained = self.complained

        date = self.date.isoformat()

        day = self.day

        delivered = self.delivered

        eligible = self.eligible

        held = self.held

        judged = self.judged

        need = self.need

        repeat = self.repeat

        share = self.share

        share_change = self.share_change

        status = self.status.value

        tiers = self.tiers



        uses = []
        for uses_item_data in self.uses:
            uses_item = uses_item_data.to_dict()
            uses.append(uses_item)



        bounce = self.bounce

        complaint = self.complaint

        healthy = self.healthy

        metric: str | Unset = UNSET
        if not isinstance(self.metric, Unset):
            metric = self.metric.value


        planned_cap = self.planned_cap

        reason: str | Unset = UNSET
        if not isinstance(self.reason, Unset):
            reason = self.reason.value


        sent = self.sent


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bounced": bounced,
            "cap": cap,
            "complained": complained,
            "date": date,
            "day": day,
            "delivered": delivered,
            "eligible": eligible,
            "held": held,
            "judged": judged,
            "need": need,
            "repeat": repeat,
            "share": share,
            "shareChange": share_change,
            "status": status,
            "tiers": tiers,
            "uses": uses,
        })
        if bounce is not UNSET:
            field_dict["bounce"] = bounce
        if complaint is not UNSET:
            field_dict["complaint"] = complaint
        if healthy is not UNSET:
            field_dict["healthy"] = healthy
        if metric is not UNSET:
            field_dict["metric"] = metric
        if planned_cap is not UNSET:
            field_dict["plannedCap"] = planned_cap
        if reason is not UNSET:
            field_dict["reason"] = reason
        if sent is not UNSET:
            field_dict["sent"] = sent

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.warmup_campaign_use import WarmupCampaignUse # noqa: PLC0415
        d = dict(src_dict)
        bounced = d.pop("bounced")

        cap = d.pop("cap")

        complained = d.pop("complained")

        date = datetime.datetime.fromisoformat(d.pop("date"))




        day = d.pop("day")

        delivered = d.pop("delivered")

        eligible = d.pop("eligible")

        held = d.pop("held")

        judged = d.pop("judged")

        need = d.pop("need")

        repeat = d.pop("repeat")

        share = d.pop("share")

        share_change = d.pop("shareChange")

        status = WarmupRowStatus(d.pop("status"))




        tiers = cast(list[str], d.pop("tiers"))


        uses = []
        _uses = d.pop("uses")
        for uses_item_data in (_uses):
            uses_item = WarmupCampaignUse.from_dict(uses_item_data)



            uses.append(uses_item)


        bounce = d.pop("bounce", UNSET)

        complaint = d.pop("complaint", UNSET)

        healthy = d.pop("healthy", UNSET)

        _metric = d.pop("metric", UNSET)
        metric: WarmupRowMetric | Unset
        if isinstance(_metric,  Unset):
            metric = UNSET
        else:
            metric = WarmupRowMetric(_metric)




        planned_cap = d.pop("plannedCap", UNSET)

        _reason = d.pop("reason", UNSET)
        reason: WarmupRowReason | Unset
        if isinstance(_reason,  Unset):
            reason = UNSET
        else:
            reason = WarmupRowReason(_reason)




        sent = d.pop("sent", UNSET)

        warmup_row = cls(
            bounced=bounced,
            cap=cap,
            complained=complained,
            date=date,
            day=day,
            delivered=delivered,
            eligible=eligible,
            held=held,
            judged=judged,
            need=need,
            repeat=repeat,
            share=share,
            share_change=share_change,
            status=status,
            tiers=tiers,
            uses=uses,
            bounce=bounce,
            complaint=complaint,
            healthy=healthy,
            metric=metric,
            planned_cap=planned_cap,
            reason=reason,
            sent=sent,
        )

        return warmup_row

