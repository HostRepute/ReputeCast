from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.whats_app_step_stats_skipped import WhatsAppStepStatsSkipped





T = TypeVar("T", bound="WhatsAppStepStats")



@_attrs_define
class WhatsAppStepStats:
    """ 
        Attributes:
            cost (float):
            delivered (int):
            failed (int):
            node_id (str):
            read (int):
            sent (int):
            skipped (WhatsAppStepStatsSkipped):
            waiting (int):
     """

    cost: float
    delivered: int
    failed: int
    node_id: str
    read: int
    sent: int
    skipped: WhatsAppStepStatsSkipped
    waiting: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.whats_app_step_stats_skipped import WhatsAppStepStatsSkipped # noqa: PLC0415
        cost = self.cost

        delivered = self.delivered

        failed = self.failed

        node_id = self.node_id

        read = self.read

        sent = self.sent

        skipped = self.skipped.to_dict()

        waiting = self.waiting


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "cost": cost,
            "delivered": delivered,
            "failed": failed,
            "nodeId": node_id,
            "read": read,
            "sent": sent,
            "skipped": skipped,
            "waiting": waiting,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.whats_app_step_stats_skipped import WhatsAppStepStatsSkipped # noqa: PLC0415
        d = dict(src_dict)
        cost = d.pop("cost")

        delivered = d.pop("delivered")

        failed = d.pop("failed")

        node_id = d.pop("nodeId")

        read = d.pop("read")

        sent = d.pop("sent")

        skipped = WhatsAppStepStatsSkipped.from_dict(d.pop("skipped"))




        waiting = d.pop("waiting")

        whats_app_step_stats = cls(
            cost=cost,
            delivered=delivered,
            failed=failed,
            node_id=node_id,
            read=read,
            sent=sent,
            skipped=skipped,
            waiting=waiting,
        )

        return whats_app_step_stats

