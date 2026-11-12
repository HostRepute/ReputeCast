from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="TransactionalStats")



@_attrs_define
class TransactionalStats:
    """ 
        Attributes:
            bounce_rate (float):
            bounced (int):
            clicked (int):
            complained (int):
            complaint_rate (float):
            deferred (int):
            delivered (int):
            delivery_rate (float):
            failed (int):
            opened (int):
            rejected (int):
            sent (int):
            median_ms (int | Unset):
            rejected_reason (str | Unset): 10 suppressed · 2 opted out
     """

    bounce_rate: float
    bounced: int
    clicked: int
    complained: int
    complaint_rate: float
    deferred: int
    delivered: int
    delivery_rate: float
    failed: int
    opened: int
    rejected: int
    sent: int
    median_ms: int | Unset = UNSET
    rejected_reason: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        bounce_rate = self.bounce_rate

        bounced = self.bounced

        clicked = self.clicked

        complained = self.complained

        complaint_rate = self.complaint_rate

        deferred = self.deferred

        delivered = self.delivered

        delivery_rate = self.delivery_rate

        failed = self.failed

        opened = self.opened

        rejected = self.rejected

        sent = self.sent

        median_ms = self.median_ms

        rejected_reason = self.rejected_reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bounceRate": bounce_rate,
            "bounced": bounced,
            "clicked": clicked,
            "complained": complained,
            "complaintRate": complaint_rate,
            "deferred": deferred,
            "delivered": delivered,
            "deliveryRate": delivery_rate,
            "failed": failed,
            "opened": opened,
            "rejected": rejected,
            "sent": sent,
        })
        if median_ms is not UNSET:
            field_dict["medianMs"] = median_ms
        if rejected_reason is not UNSET:
            field_dict["rejectedReason"] = rejected_reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        bounce_rate = d.pop("bounceRate")

        bounced = d.pop("bounced")

        clicked = d.pop("clicked")

        complained = d.pop("complained")

        complaint_rate = d.pop("complaintRate")

        deferred = d.pop("deferred")

        delivered = d.pop("delivered")

        delivery_rate = d.pop("deliveryRate")

        failed = d.pop("failed")

        opened = d.pop("opened")

        rejected = d.pop("rejected")

        sent = d.pop("sent")

        median_ms = d.pop("medianMs", UNSET)

        rejected_reason = d.pop("rejectedReason", UNSET)

        transactional_stats = cls(
            bounce_rate=bounce_rate,
            bounced=bounced,
            clicked=clicked,
            complained=complained,
            complaint_rate=complaint_rate,
            deferred=deferred,
            delivered=delivered,
            delivery_rate=delivery_rate,
            failed=failed,
            opened=opened,
            rejected=rejected,
            sent=sent,
            median_ms=median_ms,
            rejected_reason=rejected_reason,
        )

        return transactional_stats

