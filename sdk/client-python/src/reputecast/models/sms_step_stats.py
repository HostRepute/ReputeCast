from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.sms_step_stats_skipped_by import SmsStepStatsSkippedBy
  from ..models.sms_step_stats_skipped_in import SmsStepStatsSkippedIn





T = TypeVar("T", bound="SmsStepStats")



@_attrs_define
class SmsStepStats:
    """ 
        Attributes:
            clicks (int):
            cost (float):
            delivered (int):
            node_id (str):
            opt_outs (int):
            sent (int):
            skipped (int):
            skipped_by (SmsStepStatsSkippedBy):
            skipped_in (SmsStepStatsSkippedIn):
            waiting (int):
            first_at (datetime.datetime | Unset):
            last_at (datetime.datetime | Unset):
     """

    clicks: int
    cost: float
    delivered: int
    node_id: str
    opt_outs: int
    sent: int
    skipped: int
    skipped_by: SmsStepStatsSkippedBy
    skipped_in: SmsStepStatsSkippedIn
    waiting: int
    first_at: datetime.datetime | Unset = UNSET
    last_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_step_stats_skipped_by import SmsStepStatsSkippedBy # noqa: PLC0415
        from ..models.sms_step_stats_skipped_in import SmsStepStatsSkippedIn # noqa: PLC0415
        clicks = self.clicks

        cost = self.cost

        delivered = self.delivered

        node_id = self.node_id

        opt_outs = self.opt_outs

        sent = self.sent

        skipped = self.skipped

        skipped_by = self.skipped_by.to_dict()

        skipped_in = self.skipped_in.to_dict()

        waiting = self.waiting

        first_at: str | Unset = UNSET
        if not isinstance(self.first_at, Unset):
            first_at = self.first_at.isoformat()

        last_at: str | Unset = UNSET
        if not isinstance(self.last_at, Unset):
            last_at = self.last_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "clicks": clicks,
            "cost": cost,
            "delivered": delivered,
            "nodeId": node_id,
            "optOuts": opt_outs,
            "sent": sent,
            "skipped": skipped,
            "skippedBy": skipped_by,
            "skippedIn": skipped_in,
            "waiting": waiting,
        })
        if first_at is not UNSET:
            field_dict["firstAt"] = first_at
        if last_at is not UNSET:
            field_dict["lastAt"] = last_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_step_stats_skipped_by import SmsStepStatsSkippedBy # noqa: PLC0415
        from ..models.sms_step_stats_skipped_in import SmsStepStatsSkippedIn # noqa: PLC0415
        d = dict(src_dict)
        clicks = d.pop("clicks")

        cost = d.pop("cost")

        delivered = d.pop("delivered")

        node_id = d.pop("nodeId")

        opt_outs = d.pop("optOuts")

        sent = d.pop("sent")

        skipped = d.pop("skipped")

        skipped_by = SmsStepStatsSkippedBy.from_dict(d.pop("skippedBy"))




        skipped_in = SmsStepStatsSkippedIn.from_dict(d.pop("skippedIn"))




        waiting = d.pop("waiting")

        _first_at = d.pop("firstAt", UNSET)
        first_at: datetime.datetime | Unset
        if isinstance(_first_at,  Unset):
            first_at = UNSET
        else:
            first_at = datetime.datetime.fromisoformat(_first_at)




        _last_at = d.pop("lastAt", UNSET)
        last_at: datetime.datetime | Unset
        if isinstance(_last_at,  Unset):
            last_at = UNSET
        else:
            last_at = datetime.datetime.fromisoformat(_last_at)




        sms_step_stats = cls(
            clicks=clicks,
            cost=cost,
            delivered=delivered,
            node_id=node_id,
            opt_outs=opt_outs,
            sent=sent,
            skipped=skipped,
            skipped_by=skipped_by,
            skipped_in=skipped_in,
            waiting=waiting,
            first_at=first_at,
            last_at=last_at,
        )

        return sms_step_stats

